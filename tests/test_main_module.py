"""Test for running the builder as a module."""

import runpy
from unittest.mock import patch


def test_main_module_entrypoint() -> None:
    """Tests that `python -m pyvider.builder` calls the CLI."""
    with patch("pyvider.builder.cli.cli") as mock_cli:
        runpy.run_module("pyvider.builder", run_name="__main__")
    mock_cli.assert_called_once()
