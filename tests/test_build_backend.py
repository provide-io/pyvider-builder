"""Tests for the PEP 517 build backend."""

import os
from pathlib import Path

import pytest

from pyvider.builder import build_backend
from pyvider.builder.exceptions import BuildError


def test_build_backend_no_pyproject(tmp_path: Path) -> None:
    """Tests that the backend raises BuildError if pyproject.toml is missing."""
    with pytest.raises(BuildError, match="pyproject.toml not found"):
        # Simulate the build environment by changing the CWD
        os.chdir(tmp_path)
        build_backend.build_wheel(str(tmp_path))


def test_build_backend_missing_config(tmp_path: Path) -> None:
    """Tests that the backend raises BuildError for missing config sections."""
    os.chdir(tmp_path)
    pyproject = tmp_path / "pyproject.toml"

    pyproject.write_text("[project]\nname='test'")
    with pytest.raises(BuildError, match="A \\[tool.pyvider\\] section was not found"):
        build_backend.build_wheel(str(tmp_path))

    pyproject.write_text("[tool.pyvider]\nname='test'")
    with pytest.raises(
        BuildError, match="Missing 'name' or 'version' in \\[project\\] table"
    ):
        build_backend.build_wheel(str(tmp_path))
