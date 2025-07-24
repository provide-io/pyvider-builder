# Build Workflow

`pyvider-builder` offers a unified build workflow centered on `pyproject.toml`, supporting two primary methods of invocation.

## 1. Direct Build with `pyvbuild` (For Development)

This is the simplest way to build your provider binary, ideal for local development and CI.

```sh
pyvbuild package
```
This command reads `pyproject.toml` and produces a single, self-contained binary at the specified `output_path`.

## 2. Standardized Build with PEP 517 (For Integration)

This method uses standard Python build frontends, making your provider project behave like any other Python package. This is ideal for monorepos or standardized CI systems.

First, ensure your `pyproject.toml` includes the `[build-system]` and `[project.scripts]` tables:

```toml
[build-system]
requires = ["pyvider-builder"]
build-backend = "pyvider.builder.build_backend"

[project]
name = "terraform-provider-myprovider"
version = "0.1.0"
scripts = { "terraform-provider-myprovider" = "myprovider.main:serve" }
```

Then, run a standard build command:

```sh
# Build a wheel containing the provider binary
uv build

# Or install it directly into your environment
uv pip install .
```

The build frontend (e.g., `uv`) invokes the `pyvider-builder` backend, which packages the PSPF binary and places it inside a standard `.whl` file.
