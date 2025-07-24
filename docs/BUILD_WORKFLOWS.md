# `pyvider-builder` Build Workflows

The `pyvider-builder` provides a unified and flexible workflow for packaging your Python-based Terraform provider, centered around the standard `pyproject.toml` file. This approach ensures consistency whether you are developing locally or integrating into a larger CI/CD ecosystem.

## The Unified `pyproject.toml` Workflow

All configuration for your provider package is defined within a `[tool.pyvider]` section in your `pyproject.toml` file. This serves as the single source of truth for the build system.

### How it Works

1.  **Configure `pyproject.toml`**: This file is the complete "recipe" for your provider. You define the provider's identity, build dependencies, signing keys, and other metadata in one place.

    ```toml
    # pyproject.toml

    # This section is ALWAYS required for pyvider-builder.
    [tool.pyvider]
    name = "myprovider"
    output_path = "dist/terraform-provider-myprovider"
    entry_point = "myprovider.main:serve"
    python_version = "3.13" # The Python version to embed in the final binary.

    [tool.pyvider.build]
    # A list of local Python packages and PyPI dependencies.
    dependencies = [
        "./src/myprovider",
        "attrs>=23.1.0"
    ]
    # Glob patterns for files/directories to exclude from the package.
    exclude = ["**/*.pyc", ".DS_Store"]

    [tool.pyvider.signing]
    private_key_path = "keys/provider-private.key"
    public_key_path = "keys/provider-public.key"

    # --- The following sections are needed for PEP 517 builds ---

    [build-system]
    requires = ["pyvider-builder"]
    build-backend = "pyvider.builder.build_backend"

    [project]
    name = "terraform-provider-myprovider"
    version = "0.1.0"
    # The key here defines the final executable name for PEP 517 builds.
    scripts = { "terraform-provider-myprovider" = "myprovider.main:serve" }
    ```

2.  **Choose your Build Method**: Based on your goal, you can invoke the build in two ways, both of which use the same `pyproject.toml` configuration.

    *   **A) Direct Build with `pyvbuild` (Recommended for Development)**
        This is the simplest way to build your provider binary.

        ```sh
        # Ensure you have generated keys first
        pyvbuild keygen --out-dir ./keys

        # Package the provider
        pyvbuild package
        ```
        This command reads `pyproject.toml` and produces a single, self-contained binary at the specified `output_path`.

    *   **B) Standardized Build with PEP 517 (Recommended for Integration)**
        This method uses standard Python build frontends, making your provider project behave like any other Python package.

        ```sh
        # Build a wheel containing the provider binary
        uv build

        # Or install it directly into your environment
        uv pip install .
        ```
        When you run this, the build frontend (`uv` or `pip`) invokes the `pyvider-builder` backend. The backend reads its configuration from `pyproject.toml`, orchestrates the PSPF packaging, and places the final binary inside a standard `.whl` file.

### Summary

| Feature | Direct Build (`pyvbuild package`) | Standardized Build (`uv build`) |
| :--- | :--- | :--- |
| **Primary Config** | `pyproject.toml` | `pyproject.toml` |
| **Use Case** | Focused provider development & CI | Ecosystem compatibility, monorepos |
| **Invocation** | `pyvbuild package` | `uv build`, `pip install .` |
| **Output** | A single binary file | A `.whl` file containing the binary |
