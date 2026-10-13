"""Deterministic offline reasoning used by development and evaluation."""

from __future__ import annotations

import json
import re
from collections.abc import Mapping, Sequence
from typing import Any

from .base import BaseReasoner, ReasoningDecision, ToolCallDecision, tool_name


_CONTAINER_WORDS = {
    "container",
    "service",
    "the",
    "my",
    "is",
    "was",
    "crashing",
    "failed",
    "failing",
    "down",
    "restarting",
    "repeatedly",
}


class MockReasoner(BaseReasoner):
    """A small, predictable ReAct policy with no network or model dependency.

    The mock intentionally only concludes what is present in tool observations.
    It is designed for offline tests and the OOM benchmark, not as a replacement
    for a production language model.
    """

    def decide(
        self,
        user_prompt: str,
        conversation_history: Sequence[Mapping[str, Any]],
        available_tools: Sequence[Mapping[str, Any]],
    ) -> ReasoningDecision:
        tool_messages = [
            message
            for message in conversation_history
            if str(message.get("role", "")).lower() == "tool"
        ]
        observations = "\n".join(str(message.get("content", "")) for message in tool_messages)
        names = [tool_name(tool) for tool in available_tools]

        requested_name = self._requested_action(user_prompt, names)
        if requested_name:
            return self._request_tool(requested_name, user_prompt, names, len(tool_messages))

        if self._contains_oom_evidence(observations):
            return self._oom_answer(observations)

        if not tool_messages:
            first_tool = self._first_diagnostic_tool(names)
            if first_tool:
                return self._request_tool(first_tool, user_prompt, names, 0)
            return ReasoningDecision(
                thought="No diagnostic tools were provided.",
                is_final_answer=True,
                final_answer=(
                    "I cannot investigate this incident because no diagnostic tools "
                    "were provided."
                ),
            )

        # A crash investigation benefits from logs after metadata inspection, but
        # only if that tool is actually available. This gives the mock a useful,
        # deterministic second diagnostic turn without inventing observations.
        if "docker_logs" in names and not any(
            str(message.get("tool_name", "")) == "docker_logs" for message in tool_messages
        ):
            return self._request_tool("docker_logs", user_prompt, names, len(tool_messages))

        return self._insufficient_evidence_answer(observations)

    @staticmethod
    def _requested_action(prompt: str, names: Sequence[str]) -> str | None:
        lowered = prompt.lower()
        dangerous_terms = ("stop", "remove", "delete", "destroy", "restart")
        if not any(term in lowered for term in dangerous_terms):
            return None

        preferred = (
            ("docker_stop", ("stop",)),
            ("docker_remove", ("remove", "delete", "destroy")),
            ("docker_restart", ("restart",)),
        )
        for name, terms in preferred:
            if name in names and any(term in lowered for term in terms):
                return name
        return None

    @staticmethod
    def _first_diagnostic_tool(names: Sequence[str]) -> str | None:
        for preferred in ("docker_inspect", "docker_list_containers", "net_http_probe"):
            if preferred in names:
                return preferred
        return next((name for name in names if name), None)

    def _request_tool(
        self,
        name: str,
        prompt: str,
        names: Sequence[str],
        call_number: int,
    ) -> ReasoningDecision:
        if name not in names:
            return ReasoningDecision(
                thought=f"The requested tool {name!r} is unavailable.",
                is_final_answer=True,
                final_answer=f"I cannot continue because tool {name!r} is unavailable.",
            )

        arguments = self._arguments_for(name, prompt)
        call = ToolCallDecision(
            id=f"mock-call-{call_number + 1}",
            name=name,
            arguments_json=json.dumps(arguments, sort_keys=True),
        )
        return ReasoningDecision(
            thought=f"I will gather evidence with {name} before making a diagnosis.",
            requested_tool_calls=[call],
        )

    @staticmethod
    def _arguments_for(name: str, prompt: str) -> dict[str, Any]:
        if name in {
            "docker_inspect",
            "docker_logs",
            "docker_restart",
            "docker_stop",
            "docker_remove",
        }:
            return {"container_id": _container_id_from_prompt(prompt)}
        return {}

    @staticmethod
    def _contains_oom_evidence(observations: str) -> bool:
        lowered = observations.lower()
        return bool(
            re.search(r"[\"']?oom[_ ]?killed[\"']?\s*[=:]\s*true", lowered)
            or re.search(r"[\"']?(?:exit[_ ]?code|exitcode)[\"']?\s*[=:]\s*137", lowered)
            or "out of memory" in lowered
        )

    @staticmethod
    def _oom_answer(observations: str) -> ReasoningDecision:
        evidence: list[str] = []
        for line in observations.splitlines():
            lowered = line.lower()
            if any(
                marker in lowered
                for marker in ("oomkilled", "oom_killed", "exitcode", "exit_code", "out of memory")
            ):
                evidence.append(line.strip())
        evidence_text = " ".join(evidence) or "The tool observation reports an OOM condition."
        return ReasoningDecision(
            thought="The tool observation confirms memory exhaustion.",
            is_final_answer=True,
            final_answer=(
                "The container was terminated because of an out-of-memory condition. "
                f"Observed evidence: {evidence_text} "
                "Increase the container memory limit and then redeploy it."
            ),
        )

    @staticmethod
    def _insufficient_evidence_answer(observations: str) -> ReasoningDecision:
        excerpt = observations.strip()[:1200]
        return ReasoningDecision(
            thought="The available observations do not establish a root cause.",
            is_final_answer=True,
            final_answer=(
                "I cannot confirm a root cause from the available observations. "
                "I will not infer missing system state. Observed data: "
                f"{excerpt}"
            ),
        )


def _container_id_from_prompt(prompt: str) -> str:
    """Extract an explicit container/service name without inventing a real one."""

    quoted = re.search(r"['\"]([A-Za-z0-9][A-Za-z0-9_.-]*)['\"]", prompt)
    if quoted:
        return quoted.group(1)

    before_noun = re.search(
        r"\b([A-Za-z0-9][A-Za-z0-9_.-]*)\s+(?:container|service)\b",
        prompt,
        flags=re.IGNORECASE,
    )
    if before_noun and before_noun.group(1).lower() not in _CONTAINER_WORDS:
        return before_noun.group(1)

    after_noun = re.search(
        r"\b(?:container|service)\s+(?:named\s+)?([A-Za-z0-9][A-Za-z0-9_.-]*)",
        prompt,
        flags=re.IGNORECASE,
    )
    if after_noun and after_noun.group(1).lower() not in _CONTAINER_WORDS:
        return after_noun.group(1)

    # Hyphenated and underscored names are common container identifiers and are
    # less likely to be ordinary prose than a bare word such as "crashing".
    identifier = re.search(r"\b[A-Za-z0-9][A-Za-z0-9_.]*[-_][A-Za-z0-9_.-]*\b", prompt)
    if identifier:
        return identifier.group(0)

    # The gateway can still reject or refine this placeholder. It is explicit
    # about the unknown target rather than silently choosing a host resource.
    return "unknown"
