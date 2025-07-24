"""Tests for the builder's command-line interface."""

from pathlib import Path
from typing import Callable

from click.testing import CliRunner
import pytest
from pytest import MonkeyPatch

from pyvider.builder.cli import cli


@pytest.fixture
def make_sample_project(tmp_path: Path) -> Callable[[Path], Path]:
    """A factory fixture to create a sample project inside a given directory."""

    def _make_project(root_dir: Path) -> Path:
        proj_dir = root_dir / "sample_project"
        (proj_dir).mkdir()
        (proj_dir / "pyproject.toml").write_text(
            '[build-system]\nrequires=["setuptools"]\nbuild-backend="setuptools.build_meta"\n\n'
            '[project]\nname="sample"\nversion="0.1.0"\n'
        )
        (proj_dir / "main.py").write_text("def run(): print('hello')")
        return proj_dir

    return _make_project


def test_cli_package_and_verify(
    tmp_path: Path,
    private_key_pem: bytes,
    public_key_pem: bytes,
    make_sample_project: Callable[[Path], Path],
) -> None:
    """Tests the full package and verify lifecycle via the CLI."""
    runner = CliRunner()
    with runner.isolated_filesystem(temp_dir=tmp_path) as td_str:
        td = Path(td_str)
        make_sample_project(td)
        output_path = td / "dist/my-provider"
        keys_dir = td / "keys"
        keys_dir.mkdir()

        pyproject_toml_content = f"""
[tool.pyvider]
output_path = "{output_path}"
entry_point = "main:run"
[tool.pyvider.build]
dependencies = ["./sample_project"]
[tool.pyvider.signing]
private_key_path = "keys/provider-private.key"
public_key_path = "keys/provider-public.key"
"""
        (td / "pyproject.toml").write_text(pyproject_toml_content)
        (keys_dir / "provider-private.key").write_bytes(private_key_pem)
        (keys_dir / "provider-public.key").write_bytes(public_key_pem)

        package_result = runner.invoke(cli, ["package"])
        assert package_result.exit_code == 0, (
            f"Package command failed: {package_result.output}"
        )
        assert (
            "✅ Go-based cryptographic verification successful."
            in package_result.output
        )
        assert output_path.exists()


def test_cli_package_failures(tmp_path: Path) -> None:
    """Tests various failure modes of the package command."""
    runner = CliRunner()
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(cli, ["package"])
        assert result.exit_code != 0
        assert "File 'pyproject.toml' does not exist" in result.output

        Path("pyproject.toml").write_text("[project]\nname='test'")
        result = runner.invoke(cli, ["package"])
        assert result.exit_code != 0
        assert "A [tool.pyvider] section was not found" in result.output

        Path("pyproject.toml").write_text("[tool.pyvider]\nname='test'")
        result = runner.invoke(cli, ["package"])
        assert result.exit_code != 0
        assert "Missing required configuration" in result.output


def test_cli_package_missing_keys(
    tmp_path: Path, make_sample_project: Callable[[Path], Path]
) -> None:
    """Tests that packaging fails clearly if key files are missing."""
    runner = CliRunner()
    with runner.isolated_filesystem(temp_dir=tmp_path) as td_str:
        td = Path(td_str)
        make_sample_project(td)
        pyproject_toml_content = """
[tool.pyvider]
output_path = "dist/my-provider"
entry_point = "main:run"
[tool.pyvider.build]
dependencies = ["./sample_project"]
[tool.pyvider.signing]
private_key_path = "keys/provider-private.key"
public_key_path = "keys/provider-public.key"
"""
        (td / "pyproject.toml").write_text(pyproject_toml_content)
        result = runner.invoke(cli, ["package"])
        assert result.exit_code != 0
        assert "Private key not found" in result.output
        assert "Please run `pyvbuild keygen`" in result.output


def test_cli_verify_failures(tmp_path: Path) -> None:
    """Tests various failure modes of the verify command."""
    runner = CliRunner()
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(cli, ["verify"])
        assert result.exit_code != 0
        assert "Cannot find pyproject.toml to determine defaults" in result.output

        Path("pyproject.toml").write_text("""
[tool.pyvider]
output_path = "dist/provider"
[tool.pyvider.signing]
public_key_path = "keys/provider-public.key"
""")
        (Path("dist")).mkdir()
        (Path("dist/provider")).write_bytes(b"this is not a valid pspf file")
        (Path("keys")).mkdir()
        (Path("keys/provider-public.key")).touch()

        result = runner.invoke(cli, ["verify"])
        assert result.exit_code != 0
        assert "❌ Python-based verification failed" in result.output
        assert "Invalid PSPF EOF Magic" in result.output


def test_cli_clean_cache_not_found(monkeypatch: MonkeyPatch) -> None:
    """Tests the clean command when the cache directory does not exist."""
    runner = CliRunner()

    def mock_cache_dir() -> Path:
        return Path("/tmp/non-existent-pyvider-cache")

    monkeypatch.setattr("pyvider.builder.cli._get_cache_dir", mock_cache_dir)

    result = runner.invoke(cli, ["clean"])
    assert result.exit_code == 0
    assert "Cache directory not found, nothing to clean" in result.output
