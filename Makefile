# Makefile for the Pyvider Builder toolchain.
.PHONY: all venv test-go test-python clean

# Variables
VENV_DIR := .venv

# Default target
all: venv

# Create and set up the Python virtual environment.
venv:
    @echo "🐍 Setting up Python virtual environment..."
    @if [ ! -d "$(VENV_DIR)" ]; then \
        uv venv; \
    fi
    @. $(VENV_DIR)/bin/activate && uv pip install --quiet -e ".[dev]"
    @echo "✅ Virtual environment is ready."

# Run Go unit tests.
test-go:
    @echo "🧪 Running Go unit tests..."
    @cd src/pyvider/builder/go && go test -v -cover ./...
    @echo "✅ Go tests complete."

# Run Python unit and integration tests.
test-python: venv
    @echo "🐍 Running Python tests with pytest..."
    @. $(VENV_DIR)/bin/activate && pytest
    @echo "✅ Python tests complete."

# Clean build artifacts and cached Go binaries.
clean:
    @echo "🧹 Cleaning up..."
    @rm -rf dist keys *.psp
    @if [ -d "$(VENV_DIR)" ]; then \
        . $(VENV_DIR)/bin/activate && pyvbuild clean; \
    fi
    @rm -rf $(VENV_DIR)
    @find . -type f -name "*.pyc" -delete
    @find . -type d -name "__pycache__" -delete
    @echo "✅ Cleanup complete."
