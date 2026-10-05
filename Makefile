.PHONY: deps proto

deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0
	services/python-worker/.venv/Scripts/python.exe -m pip install grpcio-tools==1.64.1

proto:
	# Generate Go protobuf stubs
	services/python-worker/.venv/Scripts/python.exe -m grpc_tools.protoc \
		-I. \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/v1/agent_service.proto
	# Generate Python protobuf stubs
	services/python-worker/.venv/Scripts/python.exe -m grpc_tools.protoc \
		-I proto/v1 \
		--python_out=services/python-worker \
		--grpc_python_out=services/python-worker \
		proto/v1/agent_service.proto
