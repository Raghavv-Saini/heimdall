import json
import socket
import sys
import unittest
from pathlib import Path


WORKER_ROOT = Path(__file__).resolve().parents[1]
if str(WORKER_ROOT) not in sys.path:
    sys.path.insert(0, str(WORKER_ROOT))

import grpc  # noqa: E402

from src.proto.agent_service_pb2 import DecideRequest, Message, ToolDefinition  # noqa: E402
from src.proto.agent_service_pb2_grpc import AIWorkerServiceStub  # noqa: E402
from providers.base import format_conversation_history  # noqa: E402
from providers.mock_provider import MockReasoner  # noqa: E402
from main import build_server  # noqa: E402
from server import AIWorkerService  # noqa: E402


class WorkerTest(unittest.TestCase):
    def setUp(self):
        self.tools = [
            ToolDefinition(
                name="docker_inspect",
                description="Inspect a container",
                json_schema=json.dumps(
                    {
                        "type": "object",
                        "properties": {"container_id": {"type": "string"}},
                        "required": ["container_id"],
                    }
                ),
                risk_level="READ_ONLY",
            ),
            ToolDefinition(
                name="docker_logs",
                description="Read recent container logs",
                json_schema=json.dumps({"type": "object", "properties": {}}),
                risk_level="READ_ONLY",
            ),
        ]

    def test_conversation_history_uses_user_assistant_and_tool_roles(self):
        history = format_conversation_history(
            [
                Message(role="user", content="Why is it failing?"),
                Message(role="assistant", content="I will inspect it."),
                Message(
                    role="tool",
                    tool_call_id="call-1",
                    tool_name="docker_inspect",
                    content='{"ExitCode": 137}',
                ),
            ]
        )

        self.assertEqual(
            history,
            [
                {"role": "user", "content": "Why is it failing?"},
                {"role": "assistant", "content": "I will inspect it."},
                {
                    "role": "tool",
                    "tool_call_id": "call-1",
                    "content": '{"ExitCode": 137}',
                },
            ],
        )

    def test_mock_requests_inspection_on_first_turn(self):
        worker = AIWorkerService(reasoner=MockReasoner())
        response = worker.DecideNextStep(
            DecideRequest(
                session_id="session-1",
                user_prompt="Why is payment-processor crashing?",
                available_tools=self.tools,
            ),
            None,
        )

        self.assertFalse(response.is_final_answer)
        self.assertEqual(len(response.requested_tool_calls), 1)
        call = response.requested_tool_calls[0]
        self.assertEqual(call.name, "docker_inspect")
        self.assertEqual(json.loads(call.arguments_json), {"container_id": "payment-processor"})

    def test_grpc_server_exposes_decide_next_step(self):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
            probe.bind(("127.0.0.1", 0))
            port = probe.getsockname()[1]

        server = build_server(
            reasoner=MockReasoner(), host="127.0.0.1", port=port, max_workers=2
        )
        server.start()
        try:
            with grpc.insecure_channel(f"127.0.0.1:{port}") as channel:
                response = AIWorkerServiceStub(channel).DecideNextStep(
                    DecideRequest(
                        session_id="session-1",
                        user_prompt="Why is payment-processor crashing?",
                        available_tools=self.tools,
                    )
                )
        finally:
            server.stop(0)

        self.assertFalse(response.is_final_answer)
        self.assertEqual(response.requested_tool_calls[0].name, "docker_inspect")

    def test_mock_returns_grounded_final_answer_after_oom_observation(self):
        worker = AIWorkerService(reasoner=MockReasoner())
        response = worker.DecideNextStep(
            DecideRequest(
                session_id="session-1",
                user_prompt="Why is payment-processor crashing?",
                conversation_history=[
                    Message(role="user", content="Why is payment-processor crashing?"),
                    Message(
                        role="tool",
                        tool_call_id="call-1",
                        tool_name="docker_inspect",
                        content='{"ExitCode": 137, "OOMKilled": true}',
                    ),
                ],
                available_tools=self.tools,
            ),
            None,
        )

        self.assertTrue(response.is_final_answer)
        self.assertEqual(response.requested_tool_calls, [])
        self.assertIn("137", response.final_answer)
        self.assertIn("OOMKilled", response.final_answer)
        self.assertIn("memory", response.final_answer.lower())


if __name__ == "__main__":
    unittest.main()
