"""LiteLLM-backed provider for OpenAI-compatible and hosted models."""

from __future__ import annotations

import json
from collections.abc import Mapping, Sequence
from typing import Any, Callable

from .base import (
    SYSTEM_PROMPT,
    BaseReasoner,
    ReasoningDecision,
    ToolCallDecision,
)

try:  # Keep imports testable before dependencies are installed.
    import litellm
except ImportError:  # pragma: no cover - exercised only in incomplete environments
    litellm = None  # type: ignore[assignment]


class LiteLLMReasoner(BaseReasoner):
    """Call a configured model through LiteLLM and normalize its response."""

    def __init__(
        self,
        model: str,
        api_base: str | None = None,
        max_tokens: int = 2048,
        completion: Callable[..., Any] | None = None,
    ) -> None:
        self.model = model
        self.api_base = api_base
        self.max_tokens = max_tokens
        if completion is not None:
            self._completion = completion
        elif litellm is not None:
            self._completion = litellm.completion
        else:
            self._completion = None

    def decide(
        self,
        user_prompt: str,
        conversation_history: Sequence[Mapping[str, Any]],
        available_tools: Sequence[Mapping[str, Any]],
    ) -> ReasoningDecision:
        if self._completion is None:
            raise RuntimeError("litellm is not installed")

        messages = [
            {"role": "system", "content": SYSTEM_PROMPT},
            *[dict(message) for message in conversation_history],
        ]
        if not any(
            message.get("role") == "user" and message.get("content") == user_prompt
            for message in conversation_history
        ):
            messages.append({"role": "user", "content": user_prompt})

        request: dict[str, Any] = {
            "model": self.model,
            "messages": messages,
            "max_tokens": self.max_tokens,
            "temperature": 0,
        }
        if available_tools:
            request["tools"] = list(available_tools)
            request["tool_choice"] = "auto"
        if self.api_base:
            request["api_base"] = self.api_base

        response = self._completion(**request)
        return parse_litellm_response(response)


def parse_litellm_response(response: Any) -> ReasoningDecision:
    """Normalize common LiteLLM/OpenAI and Anthropic block responses."""

    choices = _value(response, "choices", [])
    if not choices:
        raise ValueError("LLM response did not contain a choice")
    message = _value(choices[0], "message", {})
    content, block_tool_calls = _content_and_block_tools(_value(message, "content", ""))

    raw_tool_calls = _value(message, "tool_calls", []) or []
    calls: list[ToolCallDecision] = []
    for index, raw_call in enumerate([*raw_tool_calls, *block_tool_calls]):
        function = _value(raw_call, "function", raw_call)
        name = str(_value(function, "name", ""))
        if not name:
            continue
        raw_arguments = _value(function, "arguments", _value(raw_call, "input", {}))
        arguments_json = _arguments_json(raw_arguments)
        call_id = str(_value(raw_call, "id", "") or f"llm-call-{index + 1}")
        calls.append(
            ToolCallDecision(
                id=call_id,
                name=name,
                arguments_json=arguments_json,
            )
        )

    # Some providers return a legacy singular function_call instead of tool_calls.
    legacy_call = _value(message, "function_call", None)
    if legacy_call:
        name = str(_value(legacy_call, "name", ""))
        if name:
            calls.append(
                ToolCallDecision(
                    id="llm-call-1",
                    name=name,
                    arguments_json=_arguments_json(_value(legacy_call, "arguments", {})),
                )
            )

    return ReasoningDecision(
        thought=content,
        is_final_answer=not calls,
        final_answer=content if not calls else "",
        requested_tool_calls=calls,
    )


def _value(value: Any, field: str, default: Any = None) -> Any:
    if isinstance(value, Mapping):
        return value.get(field, default)
    return getattr(value, field, default)


def _content_and_block_tools(content: Any) -> tuple[str, list[Any]]:
    if isinstance(content, str):
        return content, []
    if not isinstance(content, Sequence) or isinstance(content, (bytes, bytearray)):
        return "" if content is None else str(content), []

    text_parts: list[str] = []
    tool_calls: list[Any] = []
    for block in content:
        block_type = str(_value(block, "type", ""))
        if block_type in {"text", "output_text"}:
            text = _value(block, "text", "")
            if text:
                text_parts.append(str(text))
        elif block_type in {"tool_use", "function"}:
            tool_calls.append(block)
    return "".join(text_parts), tool_calls


def _arguments_json(arguments: Any) -> str:
    if isinstance(arguments, str):
        try:
            json.loads(arguments)
        except json.JSONDecodeError as exc:
            raise ValueError("LLM returned invalid JSON tool arguments") from exc
        return arguments
    if arguments is None:
        arguments = {}
    return json.dumps(arguments, separators=(",", ":"), sort_keys=True)
