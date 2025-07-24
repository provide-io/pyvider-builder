#!/bin/bash
#
# env.sh
#
# Standardized development environment setup script for all 'pyvider' packages.
#
# This script uses 'uv' for high-performance virtual environment creation
# and dependency management. It ensures that all related 'pyvider' sibling
# packages are installed in editable mode for a seamless development experience
# across a multi-package project.
#
# Usage: From the root of any pyvider package, run: ./env.sh
#

# --- Configuration and Setup ---

# Use colors for better readability in the terminal.
COLOR_BLUE='\033[0;34m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_NC='\033[0m' # No Color

# A small helper function to print formatted section headers.
print_header() {
    echo -e "\n${COLOR_BLUE}--- ${1} ---${COLOR_NC}"
}

# Ensure the script is run from a project root containing pyproject.toml.
if [ ! -f "pyproject.toml" ]; then
    echo -e "${COLOR_YELLOW}⚠️  Warning: 'pyproject.toml' not found.${COLOR_NC}"
    echo "Please run this script from the root directory of a pyvider package."
    exit 1
fi

PROJECT_NAME=$(basename "$(pwd)")
print_header "Setting up environment for '${PROJECT_NAME}'"

# --- 'uv' Installation ---

# Check for 'uv' and install it if it's not present. 'uv' is a modern,
# extremely fast Python package manager from Astral.
if ! command -v uv &> /dev/null; then
    print_header "🚀 Installing 'uv' package manager"
    # The official installer from astral.sh
    curl -LsSf https://astral.sh/uv/install.sh | sh

    # The uv installer places the binary in either ~/.local/bin or ~/.cargo/bin.
    # We must source the corresponding environment file to update the shell's PATH.
    # This logic robustly checks for the correct file before sourcing.
    UV_ENV_PATH_LOCAL="$HOME/.local/bin/env"
    UV_ENV_PATH_CARGO="$HOME/.cargo/env"

    if [ -f "$UV_ENV_PATH_LOCAL" ]; then
        echo "Sourcing environment from '$UV_ENV_PATH_LOCAL' to update PATH..."
        source "$UV_ENV_PATH_LOCAL"
    elif [ -f "$UV_ENV_PATH_CARGO" ]; then
        echo "Sourcing environment from '$UV_ENV_PATH_CARGO' to update PATH..."
        source "$UV_ENV_PATH_CARGO"
    else
        echo -e "${COLOR_YELLOW}⚠️  Warning: Could not find 'uv' environment file to source.${COLOR_NC}"
        echo "Please ensure '$HOME/.local/bin' or '$HOME/.cargo/bin' is in your PATH."
    fi

    echo -e "${COLOR_GREEN}✅ 'uv' installed successfully.${COLOR_NC}"
    uv --version
else
    echo -e "${COLOR_GREEN}✅ 'uv' is already installed.${COLOR_NC}"
    uv --version
fi

# --- Virtual Environment and Dependencies ---

print_header "🐍 Creating Python virtual environment with 'uv'"
uv venv

print_header "🔗 Activating the virtual environment"
# Activation is done before installing dependencies so that any console
# scripts provided by those dependencies are immediately on the PATH.
source .venv/bin/activate
echo "Virtual environment activated at '$(pwd)/.venv'"

print_header "📦 Installing base, development, and current project dependencies"
# Install dependencies defined in pyproject.toml, including the 'dev' group for the current project.
# This ensures pytest and other dev tools are available.
uv pip install .[dev]

# 'uv sync' can be used here if a uv.lock file is present and needs to be enforced.
# For now, assuming 'uv pip install .[dev]' covers the necessary dependencies for pyvider-builder.
# If a lock file (`uv.lock` or `requirements.lock`) were used, the command would be:
uv sync --all-groups # or specific extras based on the lock file content

print_header "✏️ Installing '${PROJECT_NAME}' in editable mode (if not already by above)"
# The previous `uv pip install .[dev]` might have already handled this.
# Using --no-deps and -e . ensures it's editable without reinstalling deps.
uv pip install --no-deps -e .

# --- Cross-Package Editable Installs ---

print_header "🤝 Installing sibling 'pyvider' packages in editable mode"
# This is the key logic for multi-package development. It ensures that local,
# unpublished changes in one pyvider package are used by others that depend on it.

# Correctly handle the case where we are in '/' and '..' is still '/'
PARENT_DIR=$(dirname "$(pwd)")
if [ "$PARENT_DIR" = "/" ] && [ "$(pwd)" = "/" ]; then
    echo "Running in root directory, skipping sibling search."
else
    for dir in "${PARENT_DIR}"/pyvider*; do
        # Check if the matched item is actually a directory.
        if [ -d "${dir}" ]; then
            SIBLING_PACKAGE_NAME=$(basename "${dir}")
            echo "Found sibling package: '${SIBLING_PACKAGE_NAME}'. Installing in editable mode..."
            # We use --no-deps here for a crucial reason: the main `uv sync` has
            # already resolved and installed the complete dependency graph.
            # This command's only job is to create a "link" (.pth file) to the
            # sibling's source code, effectively overriding the PyPI version.
            if ! uv pip install --no-deps -e "${dir}"; then
                echo -e "${COLOR_YELLOW}⚠️  Warning: Failed to install sibling package '${SIBLING_PACKAGE_NAME}' from '${dir}' in editable mode.${COLOR_NC}"
                echo "Attempting to continue with other packages..."
            fi
        fi
    done
fi


# --- Finalization ---

print_header "🔧 Configuring PYTHONPATH"
# Prepending the current directory's src and root to PYTHONPATH ensures
# that local modules are resolved correctly, supporting both 'src' and flat layouts.
# The ${PYTHONPATH:+:${PYTHONPATH}} syntax safely appends the existing PYTHONPATH
# only if it's already set, avoiding an "unbound variable" error.
export PYTHONPATH="${PWD}/src:${PWD}${PYTHONPATH:+:${PYTHONPATH}}"
echo "PYTHONPATH set to: ${PYTHONPATH}"

print_header "✅ Environment setup complete!"
echo -e "The '${COLOR_GREEN}${PROJECT_NAME}${COLOR_NC}' development environment is ready."
echo "The virtual environment is active in this shell."
echo "To exit, simply run 'deactivate'."
