"""
TDD tests to verify the CLI's configuration handling from pyproject.toml.
"""

from pathlib import Path
from typing import Callable
from unittest.mock import patch

from click.testing import CliRunner
import pytest

from pyvider.builder.cli import cli


@pytest.fixture
def create_sample_pyproject_toml() -> Callable[[Path], Path]:
    """
    A factory fixture that creates a sample pyproject.toml file
    inside a given directory.
    """
    def _create(root_dir: Path) -> Path:
        pyproject_content = """
[tool.pyvider]
name = "myprovider"
version = "1.2.3"
output_path = "dist/terraform-provider-myprovider"
entry_point = "myprovider.main:serve"
python_version = "3.12" # Specify a non-default version

[tool.pyvider.build]
dependencies = [
    "attrs>=23.1.0",
    "./src/myprovider",
]
exclude = ["**/*.pyc"]

[tool.pyvider.signing]
private_key_path = "keys/custom_private.key"
public_key_path = "keys/custom_public.key"
"""
        pyproject_path = root_dir / "pyproject.toml"
        pyproject_path.write_text(pyproject_content)
        # Create dummy files and dirs referenced by the config
        (root_dir / "keys").mkdir()
        (root_dir / "keys/custom_private.key").touch()
        (root_dir / "keys/custom_public.key").touch()
        (root_dir / "src/myprovider").mkdir(parents=True)
        return pyproject_path
    return _create


def test_package_command_reads_all_config_from_pyproject(
    tmp_path: Path, create_sample_pyproject_toml: Callable[[Path], Path]
):
    """
    TDD Contract: Verifies that the `package` command correctly instantiates
    the BuildOrchestrator with all configuration values read from pyproject.toml.
    """
    runner = CliRunner()
    with patch("pyvider.builder.cli.BuildOrchestrator") as MockOrchestrator:
        mock_instance = MockOrchestrator.return_value
        mock_instance.build_package.return_value = None

        with runner.isolated_filesystem(temp_dir=tmp_path) as td_str:
            td = Path(td_str)
            create_sample_pyproject_toml(td)

            # Patch the verify_command object directly within the cli module.
            # This is a targeted approach to prevent the auto-verification
            # step without interfering with the test runner's own invocation.
            with patch("pyvider.builder.cli.verify_command") as mock_verify:
                result = runner.invoke(cli, ["package", "--manifest", str(td / "pyproject.toml")])

                # The auto-verification step is only skipped, not the whole command.
                # The orchestrator call should still happen.
                assert result.exit_code == 0, f"CLI command failed: {result.output}"
                mock_verify.assert_called_once()

            # Now, verify the orchestrator was called correctly.
            MockOrchestrator.assert_called_once()
            _, kwargs = MockOrchestrator.call_args

            assert "dist/terraform-provider-myprovider" in kwargs["output_pspf_path"]
            assert "keys/custom_private.key" in kwargs["package_integrity_key_path"]
            assert "keys/custom_public.key" in kwargs["public_key_path"]
            assert kwargs["entry_point"] == "myprovider.main:serve"
            assert kwargs["python_version"] == "3.12"

            build_config = kwargs["build_config"]
            assert build_config["dependencies"] == ["attrs>=23.1.0", "./src/myprovider"]
            assert build_config["exclude"] == ["**/*.pyc"]
