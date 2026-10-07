import sys
from pathlib import Path

# Add services/python-worker to sys.path so 'src.proto' can be imported
worker_dir = Path(__file__).resolve().parent.parent
if str(worker_dir) not in sys.path:
    sys.path.insert(0, str(worker_dir))


def test_proto_and_grpc_imports():
    """Verify that generated proto and gRPC stubs can be cleanly imported as part of the package."""
    from src.proto import agent_service_pb2
    from src.proto import agent_service_pb2_grpc

    assert hasattr(agent_service_pb2, "DecideRequest")
    assert hasattr(agent_service_pb2, "DecideResponse")
    assert hasattr(agent_service_pb2, "Message")
    assert hasattr(agent_service_pb2, "ToolDefinition")
    assert hasattr(agent_service_pb2, "ToolCall")
    assert hasattr(agent_service_pb2_grpc, "AIWorkerServiceStub")
    assert hasattr(agent_service_pb2_grpc, "AIWorkerServiceServicer")
    assert hasattr(agent_service_pb2_grpc, "add_AIWorkerServiceServicer_to_server")


def test_decide_request_serialization():
    """Verify that DecideRequest can be created, serialized, and deserialized."""
    from src.proto import agent_service_pb2

    msg1 = agent_service_pb2.Message(
        role="user",
        content="why is my container crashing?",
        tool_call_id="",
        tool_name="",
    )
    msg2 = agent_service_pb2.Message(
        role="tool",
        content='{"ExitCode": 137}',
        tool_call_id="call-1",
        tool_name="docker_inspect",
    )
    tool = agent_service_pb2.ToolDefinition(
        name="docker_inspect",
        description="Inspect container state",
        json_schema='{"type": "object"}',
        risk_level="READ_ONLY",
    )

    req = agent_service_pb2.DecideRequest(
        session_id="session-xyz",
        user_prompt="why is my container crashing?",
        conversation_history=[msg1, msg2],
        available_tools=[tool],
    )

    data = req.SerializeToString()
    assert isinstance(data, bytes)

    deserialized = agent_service_pb2.DecideRequest()
    deserialized.ParseFromString(data)

    assert deserialized.session_id == "session-xyz"
    assert deserialized.user_prompt == "why is my container crashing?"
    assert len(deserialized.conversation_history) == 2
    assert deserialized.conversation_history[0].role == "user"
    assert deserialized.conversation_history[1].role == "tool"
    assert deserialized.conversation_history[1].tool_call_id == "call-1"
    assert deserialized.conversation_history[1].tool_name == "docker_inspect"
    assert len(deserialized.available_tools) == 1
    assert deserialized.available_tools[0].name == "docker_inspect"
    assert deserialized.available_tools[0].risk_level == "READ_ONLY"


def test_decide_response_serialization():
    """Verify that DecideResponse can be created, serialized, and deserialized."""
    from src.proto import agent_service_pb2

    tool_call = agent_service_pb2.ToolCall(
        id="call-42",
        name="docker_restart",
        arguments_json='{"container_id": "web-api"}',
    )
    resp = agent_service_pb2.DecideResponse(
        thought="Restarting container web-api",
        is_final_answer=False,
        final_answer="",
        requested_tool_calls=[tool_call],
    )

    data = resp.SerializeToString()
    assert isinstance(data, bytes)

    deserialized = agent_service_pb2.DecideResponse()
    deserialized.ParseFromString(data)

    assert deserialized.thought == "Restarting container web-api"
    assert deserialized.is_final_answer is False
    assert len(deserialized.requested_tool_calls) == 1
    assert deserialized.requested_tool_calls[0].id == "call-42"
    assert deserialized.requested_tool_calls[0].name == "docker_restart"
    assert deserialized.requested_tool_calls[0].arguments_json == '{"container_id": "web-api"}'


def test_decide_response_final_answer():
    """Verify DecideResponse final diagnosis serialization."""
    from src.proto import agent_service_pb2

    resp = agent_service_pb2.DecideResponse(
        thought="Diagnosis complete.",
        is_final_answer=True,
        final_answer="### Root Cause\nOOMKilled container.",
        requested_tool_calls=[],
    )

    data = resp.SerializeToString()
    deserialized = agent_service_pb2.DecideResponse()
    deserialized.ParseFromString(data)

    assert deserialized.is_final_answer is True
    assert "Root Cause" in deserialized.final_answer
    assert len(deserialized.requested_tool_calls) == 0
