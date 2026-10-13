"""Shared reasoning models and protocol-to-provider conversions."""

from __future__ import annotations

import json
from abc import ABC, abstractmethod
from collections.abc import Mapping, Sequence
from typing import Any

from pydantic import BaseModel, Field


SYSTEM_PROMPT = """You are Heimdall, an expert DevOps investigation agent.

Investigate infrastructure and container incidents one diagnostic step at a time.
Ground every conclusion in observations returned by tools. Never invent logs, exit
codes, container names, or other system state. Request a diagnostic tool when more
evidence is needed. When the evidence is sufficient, return a concise Markdown
root-cause diagnosis with the observed evidence and a safe remediation suggestion.
Do not execute tools yourself; request them using their declared schemas.
"""


class ToolCallDecision(BaseModel):
    """A structured tool request returned by a reasoning provider."""

    id: str
    name: str
    arguments_json: str


class ReasoningDecision(BaseModel):
    """Provider-independent result for one ReAct turn."""

    thought: str = ""
    is_final_answer: bool = False
    final_answer: str = ""
    requested_tool_calls: list[ToolCallDecision] = Field(default_factory=list)


class BaseReasoner(ABC):
    """Public seam implemented by live and deterministic reasoning providers."""

    @abstractmethod
    def decide(
        self,
        user_prompt: str,
        conversation_history: Sequence[Mapping[str, Any]],
        available_tools: Sequence[Mapping[str, Any]],
    ) -> ReasoningDecision:
        """Choose the next tool request or return a final answer."""


def _message_value(message: Any, field: str, default: Any = "") -> Any:
    if isinstance(message, Mapping):
        return message.get(field, default)
    return getattr(message, field, default)


def format_conversation_history(messages: Sequence[Any]) -> list[dict[str, Any]]:
    """Convert protobuf messages to the common user/assistant/tool format.

    LiteLLM accepts the OpenAI-compatible message shape. Tool observations retain
    their call ID so providers can associate an observation with the requested
    call. Unknown roles are treated as user content rather than being allowed to
    introduce an uncontrolled provider message type.
    """

    formatted: list[dict[str, Any]] = []
    for message in messages:
        role = str(_message_value(message, "role", "user")).lower()
        if role not in {"user", "assistant", "tool"}:
            role = "user"

        item: dict[str, Any] = {
            "role": role,
            "content": str(_message_value(message, "content", "")),
        }
        if role == "tool":
            tool_call_id = str(_message_value(message, "tool_call_id", ""))
            if tool_call_id:
                item["tool_call_id"] = tool_call_id
        formatted.append(item)
    return formatted


def format_tool_definitions(tools: Sequence[Any]) -> list[dict[str, Any]]:
    """Convert protobuf tool definitions to LiteLLM's function-tool format."""

    formatted: list[dict[str, Any]] = []
    for tool in tools:
        name = str(_message_value(tool, "name", ""))
        if not name:
            raise ValueError("tool definition is missing a name")

        raw_schema = str(_message_value(tool, "json_schema", "") or "{}")
        try:
            parameters = json.loads(raw_schema)
        except json.JSONDecodeError as exc:
            raise ValueError(f"tool {name!r} has invalid JSON schema") from exc
        if not isinstance(parameters, dict):
            raise ValueError(f"tool {name!r} schema must be a JSON object")

        formatted.append(
            {
                "type": "function",
                "function": {
                    "name": name,
                    "description": str(_message_value(tool, "description", "")),
                    "parameters": parameters,
                },
            }
        )
    return formatted


def tool_name(tool: Mapping[str, Any]) -> str:
    """Return a normalized function name from a formatted tool definition."""

    function = tool.get("function", {})
    if isinstance(function, Mapping):
        return str(function.get("name", ""))
    return ""


def tool_description(tool: Mapping[str, Any]) -> str:
    function = tool.get("function", {})
    if isinstance(function, Mapping):
        return str(function.get("description", ""))
    return ""
