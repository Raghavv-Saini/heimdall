VENV_DIR := services/python-worker/.venv
GO_BIN_DIR := $(shell go env GOPATH 2>/dev/null)/bin
export PATH := $(GO_BIN_DIR):$(PATH)

ifeq ($(OS),Windows_NT)
    VENV_PYTHON := $(VENV_DIR)/Scripts/python.exe
    SYSTEM_PYTHON ?= python
else
    VENV_PYTHON := $(VENV_DIR)/bin/python
    SYSTEM_PYTHON ?= $(shell command -v python3 2>/dev/null || command -v python 2>/dev/null)
endif

# Use virtualenv python if it exists, otherwise fall back to system python
PYTHON ?= $(if $(wildcard $(VENV_PYTHON)),$(VENV_PYTHON),$(SYSTEM_PYTHON))

PREFIX ?= $(HOME)/.local
BIN_DIR ?= $(PREFIX)/bin

.PHONY: all deps proto build test eval install clean

all: proto build test

deps:
	@if [ ! -d "$(VENV_DIR)" ]; then \
		echo "Bootstrapping Python virtual environment in $(VENV_DIR)..."; \
		$(SYSTEM_PYTHON) -m venv $(VENV_DIR) 2>/dev/null || $(SYSTEM_PYTHON) -m venv --without-pip $(VENV_DIR); \
	fi
	@if ! $(VENV_PYTHON) -m pip --version >/dev/null 2>&1; then \
		echo "Bootstrapping pip into virtual environment..."; \
		curl -sSL https://bootstrap.pypa.io/get-pip.py | $(VENV_PYTHON); \
	fi
	@if command -v go >/dev/null 2>&1; then \
		echo "Installing Go protobuf plugins..."; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0; \
	fi
	@echo "Installing Python dependencies..."
	$(VENV_PYTHON) -m pip install --upgrade pip
	$(VENV_PYTHON) -m pip install grpcio-tools==1.64.1 pytest==8.1.0

proto:
	@mkdir -p proto/v1
	@mkdir -p services/python-worker/src/proto
	# Generate Go protobuf stubs
	$(PYTHON) -m grpc_tools.protoc \
		-I. \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/v1/agent_service.proto
	# Generate Python protobuf stubs
	$(PYTHON) -m grpc_tools.protoc \
		-I proto/v1 \
		--python_out=services/python-worker/src/proto \
		--grpc_python_out=services/python-worker/src/proto \
		proto/v1/agent_service.proto
	# Fix package-relative import in generated Python gRPC stub
	$(PYTHON) -c "import re; p = 'services/python-worker/src/proto/agent_service_pb2_grpc.py'; c = open(p).read(); c = re.sub(r'^import agent_service_pb2 as', 'from . import agent_service_pb2 as', c, flags=re.MULTILINE); open(p, 'w').write(c)"

build:
	@if [ -d "cmd/heimdall" ]; then \
		echo "Building host CLI binary..."; \
		mkdir -p bin && go build -o bin/heimdall ./cmd/heimdall && ln -sf heimdall bin/hml; \
	else \
		echo "Verifying Go packages compilation..."; \
		go build ./proto/...; \
	fi

test:
	@echo "Running Go tests..."
	go test -v ./...
	@echo "Running Python worker tests..."
	$(PYTHON) -m pytest services/python-worker/tests

eval:
	@if [ -f "tests/evals/runner.py" ]; then \
		$(PYTHON) tests/evals/runner.py; \
	else \
		echo "Eval benchmark suite (tests/evals/runner.py) not yet implemented (Ticket 06)."; \
	fi

install:
	@if [ -f "bin/heimdall" ]; then \
		mkdir -p $(DESTDIR)$(BIN_DIR); \
		cp bin/heimdall $(DESTDIR)$(BIN_DIR)/heimdall; \
		ln -sf heimdall $(DESTDIR)$(BIN_DIR)/hml; \
		echo "Installed heimdall and hml into $(DESTDIR)$(BIN_DIR)"; \
	else \
		echo "Binary bin/heimdall not found. Run 'make build' first once CLI is implemented."; \
	fi

clean:
	rm -rf bin/
	rm -rf services/python-worker/src/proto/__pycache__
	rm -rf services/python-worker/tests/__pycache__
	rm -rf services/python-worker/__pycache__
	rm -rf .pytest_cache
