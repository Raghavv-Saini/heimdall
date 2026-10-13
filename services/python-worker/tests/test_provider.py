import json
import sys
import unittest
from pathlib import Path


WORKER_ROOT = Path(__file__).resolve().parents[1]
if str(WORKER_ROOT) not in sys.path:
    sys.path.insert(0, str(WORKER_ROOT))

from src.proto.agent_service_pb2 import Message  # noqa: E402
from providers.litellm_provider import LiteLLMReasoner  # noqa: E402
from server import create_reasoner  # noqa: E402


class LiteLLMProviderTest(unittest.TestCase):
    def test_tool_call_response_is_normalized_and_request_is_provider_formatted(self):
        captured = {}

        def completion(**request):
            captured.update(request)
            return {
                "choices": [
                    {
                        "message": {
                            "content": "I need container metadata first.",
                            "tool_calls": [
                                {
                                    "id": "call-1",
                                    "function": {
                                        "name": "docker_inspect",
                                        "arguments": '{"container_id":"payment-processor"}',
                                    },
                                }
                            ],
                        }
                    }
                ]
            }

        reasoner = LiteLLMReasoner(model="openai/test-model", completion=completion)
        decision = reasoner.decide(
            "Why is payment-processor crashing?",
            [
                {"role": "user", "content": "Why is payment-processor crashing?"},
                {
                    "role": "tool",
                    "tool_call_id": "previous-call",
                    "content": "previous observation",
                },
            ],
            [
                {
                    "type": "function",
                    "function": {
                        "name": "docker_inspect",
                        "description": "Inspect a container",
                        "parameters": {"type": "object"},
                    },
                }
            ],
        )

        self.assertFalse(decision.is_final_answer)
        self.assertEqual(decision.requested_tool_calls[0].name, "docker_inspect")
        self.assertEqual(
            json.loads(decision.requested_tool_calls[0].arguments_json),
            {"container_id": "payment-processor"},
        )
        self.assertEqual(captured["model"], "openai/test-model")
        self.assertEqual(captured["messages"][1]["role"], "user")
        self.assertEqual(captured["messages"][2]["role"], "tool")
        self.assertEqual(captured["tools"][0]["type"], "function")

    def test_text_only_response_becomes_final_answer(self):
        reasoner = LiteLLMReasoner(
            model="openai/test-model",
            completion=lambda **_: {
                "choices": [{"message": {"content": "The service is healthy."}}]
            },
        )

        decision = reasoner.decide("Investigate", [], [])

        self.assertTrue(decision.is_final_answer)
        self.assertEqual(decision.final_answer, "The service is healthy.")
        self.assertEqual(decision.requested_tool_calls, [])


class ProviderSelectionTest(unittest.TestCase):
    def test_missing_credentials_selects_offline_mock(self):
        reasoner = create_reasoner({"LLM_MODEL": "openai/test-model"})
        self.assertEqual(type(reasoner).__name__, "MockReasoner")

    def test_explicit_mock_flag_overrides_provider_credentials(self):
        reasoner = create_reasoner(
            {
                "OPENAI_API_KEY": "test-key",
                "HEIMDALL_MOCK_REASONING": "true",
            }
        )
        self.assertEqual(type(reasoner).__name__, "MockReasoner")

    def test_provider_credentials_select_litellm(self):
        reasoner = create_reasoner(
            {
                "OPENAI_API_KEY": "test-key",
                "LLM_MODEL": "openai/test-model",
                "OPENAI_API_BASE": "http://localhost:11434/v1",
            }
        )
        self.assertEqual(type(reasoner).__name__, "LiteLLMReasoner")
        self.assertEqual(reasoner.model, "openai/test-model")
        self.assertEqual(reasoner.api_base, "http://localhost:11434/v1")


if __name__ == "__main__":
    unittest.main()
