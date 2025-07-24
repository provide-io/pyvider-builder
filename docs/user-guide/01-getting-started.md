# Getting Started

This guide will walk you through building your first provider with `pyvider-builder`.

## Prerequisites

- Python 3.12+
- Go 1.22+
- `uv` package manager

## Installation

Install `pyvider-builder` from PyPI:

```sh
uv pip install pyvider-builder
```

## Quick Start

1.  **Configure `pyproject.toml`**: In your provider project, configure the `[tool.pyvider]` section. This is the single source of truth for the build.

    ```toml
    # pyproject.toml
    [tool.pyvider]
    output_path = "dist/terraform-provider-myprovider"
    entry_point = "myprovider.main:serve"
    python_version = "3.13"

    [tool.pyvider.build]
    dependencies = [
        "./src/myprovider", # Your local provider code
        "attrs>=23.1.0"   # Any PyPI dependencies
    ]

    [tool.pyvider.signing]
    private_key_path = "keys/provider-private.key"
    public_key_path = "keys/provider-public.key"
    ```

2.  **Generate Signing Keys**:
    ```sh
    pyvbuild keygen --out-dir ./keys
    ```

3.  **Package the Provider**:
    ```sh
    pyvbuild package
    ```

This will produce your self-contained provider binary at `dist/terraform-provider-myprovider`, ready for local testing with Terraform.
