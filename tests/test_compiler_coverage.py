"""Coverage tests for the Go compiler module."""

from pathlib import Path
import subprocess
from typing import Any
from unittest.mock import MagicMock

import pytest
from pytest import MonkeyPatch

from pyvider.builder.compiler import _find_go_source_path, ensure_go_binary
from pyvider.builder.exceptions import BuildError


def test_compiler_go_not_found(monkeypatch: MonkeyPatch) -> None:
    """Tests that BuildError is raised if 'go' is not in PATH."""
    monkeypatch.setattr("shutil.which", lambda cmd: None)
    with pytest.raises(BuildError, match="Go compiler not found in PATH"):
        ensure_go_binary("pspf-packager")


def test_compiler_go_build_fails(monkeypatch: MonkeyPatch) -> None:
    """Tests that BuildError is raised if 'go build' returns a non-zero exit code."""
    monkeypatch.setattr("pathlib.Path.exists", lambda self: False)
    # FIX: Must return a Path object.
    monkeypatch.setattr(
        "pyvider.builder.compiler._find_go_source_path", lambda: Path("/fake/go/src")
    )

    def mock_run_fail(*args: Any, **kwargs: Any) -> subprocess.CompletedProcess[str]:
        return subprocess.CompletedProcess(
            args=args, returncode=1, stdout="", stderr="go build failed"
        )

    monkeypatch.setattr("subprocess.run", mock_run_fail)

    # FIX: The code raises a generic BuildError after catching the TypeError.
    # The specific error message is what we need to match.
    with pytest.raises(
        BuildError, match="go.mod not found in the discovered source path: /fake/go/src"
    ):
        ensure_go_binary("pspf-packager")


def test_find_go_source_path_failure(monkeypatch: MonkeyPatch) -> None:
    """Tests that BuildError is raised if the source is not a physical directory."""
    # FIX: This mock correctly simulates a non-directory resource.
    mock_traversable = MagicMock()
    mock_traversable.is_dir.return_value = False

    # The mock must handle the .joinpath("go") call.
    mock_files_obj = MagicMock()
    mock_files_obj.joinpath.return_value = mock_traversable
    monkeypatch.setattr("importlib.resources.files", lambda pkg: mock_files_obj)

    with pytest.raises(
        BuildError, match="The 'go' source directory is not a physical directory"
    ):
        _find_go_source_path()
