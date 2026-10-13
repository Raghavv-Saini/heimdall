"""Entrypoint for the Heimdall Python AI worker gRPC server."""

from __future__ import annotations

import concurrent.futures

import grpc

from src.proto import agent_service_pb2_grpc
from server import AIWorkerService, AIWorkerServiceServicer, create_reasoner

DEFAULT_HOST = "[::]"
DEFAULT_PORT = 50052


def build_server(
    reasoner=None,
    host: str = DEFAULT_HOST,
    port: int = DEFAULT_PORT,
    max_workers: int = 10,
):
    """Build, register, and bind the worker server without starting it."""

    server = grpc.server(concurrent.futures.ThreadPoolExecutor(max_workers=max_workers))
    service = AIWorkerService(reasoner=reasoner or create_reasoner())
    agent_service_pb2_grpc.add_AIWorkerServiceServicer_to_server(service, server)
    server.add_insecure_port(f"{host}:{port}")
    return server


def serve() -> None:
    server = build_server()
    server.start()
    print(f"Heimdall Python AI Worker running on gRPC port {DEFAULT_PORT}...", flush=True)
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=0)


# Re-export the service for callers and tests that use main.py as the service entrypoint.
__all__ = ["AIWorkerService", "AIWorkerServiceServicer", "build_server", "serve"]


if __name__ == "__main__":
    serve()
