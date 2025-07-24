"""
Tests for uncovered code paths in the pyvider-builder to improve coverage.
"""

import subprocess
from typing import Any, Never

from click.testing import CliRunner
import pytest
from pytest import MonkeyPatch

from pyvider.builder.cli import cli
from pyvider.builder.exceptions import BuildError
import pyvider.builder.models


def test_compiler_source_not_found(monkeypatch: MonkeyPatch) -> None:
    """
    Tests that the compiler raises BuildError if the Go source directory cannot be found.
    """
    from pyvider.builder import compiler

    # First, ensure the cache is clean so ensure_go_binary doesn't find a pre-built binary.
    runner = CliRunner()
    clean_result = runner.invoke(cli, ["clean"])
    assert clean_result.exit_code == 0, (
        "The 'clean' command must succeed for this test to be valid."
    )

    # This mock correctly simulates the case where neither the editable nor wheel path is found.
    def mock_find_fail() -> Never:
        raise BuildError("Bundled Go source directory not found.")

    monkeypatch.setattr(compiler, "_find_go_source_path", mock_find_fail)

    # Now, calling ensure_go_binary will be forced to use the mocked function and fail.
    with pytest.raises(BuildError, match="Bundled Go source directory not found"):
        compiler.ensure_go_binary("pspf-packager")


def test_cli_keygen_subprocess_failure(monkeypatch: MonkeyPatch) -> None:
    """
    Tests the `keygen` CLI command's error handling when the subprocess fails.
    This test is now fully isolated and does not depend on the compiler cache.
    """
    # FIX: Mock the entire compiler function. We are not testing the compiler here,
    # we are testing the CLI's error handling. This prevents any real compilation attempt.
    monkeypatch.setattr(
        "pyvider.builder.cli.ensure_go_binary",
        lambda tool_name: "/dummy/path/to/pspf-packager",
    )

    # Mock the subprocess.run call to simulate a failure
    def mock_run_fail(*args: Any, **kwargs: Any) -> Never:
        raise subprocess.CalledProcessError(
            1, cmd=args[0], stderr="Go tool failed spectacularly"
        )

    monkeypatch.setattr("pyvider.builder.cli.subprocess.run", mock_run_fail)

    runner = CliRunner()
    result = runner.invoke(cli, ["keygen"])

    assert result.exit_code != 0, "CLI should exit with a non-zero code on failure."
    assert "❌ Keygen failed:" in result.output
    assert "Go tool failed spectacularly" in result.output


def test_footer_model_validation() -> None:
    """
    Tests the validation logic within the PspfFooter model, using correct field names.
    """
    valid_args = {
        "uv_binary_offset": 10,
        "uv_binary_size": 20,
        "python_install_tgz_offset": 30,
        "python_install_tgz_size": 40,
        "metadata_tgz_offset": 70,
        "metadata_tgz_size": 50,
        "payload_tgz_offset": 120,
        "payload_tgz_size": 60,
        "package_signature_offset": 180,
        "package_signature_size": 256,
        "public_key_pem_offset": 436,
        "public_key_pem_size": 800,
    }
    footer_with_bad_version = pyvider.builder.models.PspfFooter(
        **valid_args, pspf_version=0x9999
    )
    packed_bad_version_footer = footer_with_bad_version.pack()

    with pytest.raises(ValueError, match="Unexpected PSPF version"):
        pyvider.builder.models.PspfFooter.unpack(packed_bad_version_footer)
