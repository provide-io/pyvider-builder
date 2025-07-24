# Pyvider Builder (`pyvbuild`)

`pyvider.builder` is the official toolchain for packaging Python-based Terraform providers built with the [Pyvider framework](https://github.com/provide-io/pyvider). It solves the complex problem of distributing a Python application as a single, self-contained, secure, and performant binary that is compatible with Terraform's plugin model.

The tool produces a **Progressive Secure Package Format (PSPF)** file, which is a Go-based launcher that embeds a Python runtime, the `uv` package manager, and all of your provider's Python code and dependencies. When Terraform runs the provider, the launcher creates a fully isolated environment on-the-fly, ensuring your provider runs correctly and consistently everywhere.

## Features

- **Zero Dependencies**: End-users only need Terraform. The packaged provider binary contains everything else it needs to run.
- **Secure by Default**: Packages are cryptographically signed using RSA-PSS to ensure integrity and authenticity.
- **Performant**: The Go launcher is fast, and subsequent runs are cached for near-native startup times.
- **Unified Build Workflow**: Supports a standardized build process driven by `pyproject.toml`, compatible with both direct CLI usage and standard Python build frontends (PEP 517).

**For a detailed explanation of the build process, please see [BUILD_WORKFLOWS.md](./BUILD_WORKFLOWS.md).**

## Quick Start

1.  **Install `pyvider.builder` and Go**:
    ```sh
    uv pip install pyvider-builder
    # Ensure you have Go 1.22+ installed and in your PATH
    ```

2.  **Configure your provider's `pyproject.toml`**:
    ```toml
    # pyproject.toml
    [tool.pyvider]
    name = "myprovider"
    output_path = "dist/terraform-provider-myprovider"
    entry_point = "myprovider.main:serve"
    python_version = "3.13" # Specify the embedded Python version

    [tool.pyvider.build]
    dependencies = ["./src/myprovider", "attrs"]

    [tool.pyvider.signing]
    private_key_path = "keys/provider-private.key"
    public_key_path = "keys/provider-public.key"

    # For PEP 517 builds, you also need:
    [build-system]
    requires = ["pyvider-builder"]
    build-backend = "pyvider.builder.build_backend"

    [project]
    name = "terraform-provider-myprovider"
    version = "0.1.0"
    # This script name becomes the final executable name.
    scripts = { "terraform-provider-myprovider" = "myprovider.main:serve" }
    ```

3.  **Generate Signing Keys**:
    ```sh
    pyvbuild keygen --out-dir ./keys
    ```

4.  **Package your provider**:
    ```sh
    pyvbuild package
    ```

This will produce your self-contained provider binary at `dist/terraform-provider-myprovider`.
