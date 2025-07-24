"""
On-demand compiler for the Go binaries bundled with pyvider-builder.
"""

import importlib.resources
from pathlib import Path
import shutil
import subprocess

import click

from .exceptions import BuildError


def _get_cache_dir() -> Path:
    """Returns the user-specific cache directory for pyvider-builder binaries."""
    cache_dir = Path.home() / ".cache" / "pyvider-builder"
    cache_dir.mkdir(parents=True, exist_ok=True)
    return cache_dir


def _find_go_source_path() -> Path:
    """
    Locates the bundled 'go' source directory inside the installed package.
    """
    try:
        go_src_traversable = importlib.resources.files("pyvider.builder").joinpath("go")

        if not go_src_traversable.is_dir():
            raise BuildError(
                "The 'go' source directory is not a physical directory. "
                "This can happen if the package is installed from a zip/egg."
            )

        return Path(str(go_src_traversable))

    except (ImportError, AttributeError, FileNotFoundError) as e:
        raise BuildError(
            f"Could not find bundled Go source directory via importlib.resources: {e}"
        ) from e


def ensure_go_binary(tool_name: str) -> Path:
    """
    Ensures a Go binary is compiled and ready, returning its path.
    """
    if not shutil.which("go"):
        raise BuildError("Go compiler not found in PATH. Please install Go.")

    bin_cache_dir = _get_cache_dir() / "bin"
    bin_cache_dir.mkdir(exist_ok=True)
    binary_path = bin_cache_dir / tool_name

    if binary_path.exists():
        return binary_path

    click.secho(
        f"Go binary '{tool_name}' not found in cache. Compiling...", fg="yellow"
    )

    try:
        go_module_root = _find_go_source_path()

        if not (go_module_root / "go.mod").exists():
            raise BuildError(
                f"go.mod not found in the discovered source path: {go_module_root}"
            )

        # Add -buildvcs=false to prevent Go from trying to find a .git directory.
        # This is essential for building in clean environments like Docker or CI.
        cmd = [
            "go",
            "build",
            "-buildvcs=false",
            "-o",
            str(binary_path),
            f"./{tool_name}",
        ]
        result = subprocess.run(
            cmd, cwd=go_module_root, capture_output=True, text=True, check=False
        )

        if result.returncode != 0:
            raise BuildError(
                f"Failed to compile Go binary '{tool_name}'.\nStderr: {result.stderr.strip()}"
            )

        click.secho(
            f"Successfully compiled '{tool_name}' to '{binary_path}'.", fg="green"
        )
        return binary_path

    except Exception as e:
        if isinstance(e, BuildError):
            raise
        raise BuildError(
            f"An unexpected error occurred during Go compilation: {e}"
        ) from e
