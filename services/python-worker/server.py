"""gRPC service adapter for the Heimdall reasoning providers."""

from __future__ import annotations

import os
from collections.abc import Mapping
from typing import Any

from src.proto import agent_service_pb2, agent_service_pb2_grpc

from providers.base import (
    BaseReasoner,
    format_conversation_history,
    format_tool_definitions,
)
from providers.litellm_provider import LiteLLMReasoner
from providers.mock_provider import MockReasoner


DEFAULT_MODEL = "anthropic/claude-3-5-sonnet-20241022"
_PROVIDER_KEYS = (
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY",
    "GEMINI_API_KEY",
    "GOOGLE_API_KEY",
    "GROQ_API_KEY",
)


class AIWorkerServiceServicer(agent_service_pb2_grpc.AIWorkerServiceServicer):
    """Translate protobuf requests into provider-neutral reasoning decisions."""

    def __init__(self, reasoner: BaseReasoner | None = None, environ: Mapping[str, str] | None = None):
        self._environment = dict(environ or os.environ)
        self.reasoner = reasoner or create_reasoner(self._environment)

    def DecideNextStep(self, request, context):  # noqa: N802 - generated RPC name
        try:
            history = format_conversation_history(request.conversation_history)
            tools = format_tool_definitions(request.available_tools)
            decision = self.reasoner.decide(request.user_prompt, history, tools)
            return _decision_to_proto(decision)
        except Exception as exc:  # Keep provider failures inside the RPC contract.
            return agent_service_pb2.DecideResponse(
                thought=f"Reasoning provider error: {type(exc).__name__}.",
                is_final_answer=True,
                final_answer=(
                    "Heimdall could not produce a reasoning step. Check the worker "
                    "provider configuration and model credentials."
                ),
            )


def create_reasoner(environ: Mapping[str, str] | None = None) -> BaseReasoner:
    """Select live or offline reasoning from environment configuration."""

    environment = dict(environ or os.environ)
    if _truthy(environment.get("HEIMDALL_MOCK_REASONING")) or not _has_provider_config(
        environment
    ):
        return MockReasoner()

    return LiteLLMReasoner(
        model=environment.get("LLM_MODEL", DEFAULT_MODEL),
        api_base=environment.get("OPENAI_API_BASE") or None,
    )


def _has_provider_config(environment: Mapping[str, str]) -> bool:
    return any(environment.get(key) for key in _PROVIDER_KEYS) or bool(
        environment.get("OPENAI_API_BASE")
    )


def _truthy(value: str | None) -> bool:
    return str(value or "").strip().lower() in {"1", "true", "yes", "on"}


# Keep the shorter name as a compatibility alias for callers that use the
# service name rather than the generated gRPC base class name.
AIWorkerService = AIWorkerServiceServicer


def _decision_to_proto(decision: Any) -> Any:
    response = agent_service_pb2.DecideResponse(  # type: ignore[attr-defined]
        thought=decision.thought,
        is_final_answer=decision.is_final_answer,
        final_answer=decision.final_answer,
    )
    response.requested_tool_calls.extend(
        agent_service_pb2.ToolCall(  # type: ignore[attr-defined]
            id=call.id,
            name=call.name,
            arguments_json=call.arguments_json,
        )
        for call in decision.requested_tool_calls
    )
    return response
