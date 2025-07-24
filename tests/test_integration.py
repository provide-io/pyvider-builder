"""
Integration tests that execute compiled PSPF packages.
"""

import os
from pathlib import Path
import stat
import subprocess
from typing import Callable

from click.testing import CliRunner
import pytest

from pyvider.builder.cli import cli


@pytest.fixture
def make_sample_project(tmp_path: Path) -> Callable[[Path], Path]:
    """A factory fixture to create a sample project inside a given directory."""

    def _make_project(root_dir: Path) -> Path:
        proj_dir = root_dir / "sample_project"
        src_dir = proj_dir / "src"
        pkg_dir = src_dir / "my_sample_pkg"
        pkg_dir.mkdir(parents=True)

        (proj_dir / "pyproject.toml").write_text(
            '[build-system]\nrequires=["setuptools"]\nbuild-backend="setuptools.build_meta"\n\n'
            '[project]\nname="my-sample-pkg"\nversion="0.1.0"\n'
        )
        (pkg_dir / "__init__.py").touch()
        (pkg_dir / "main.py").write_text(
            "import asyncio\n"
            "async def serve():\n"
            "    print('--- My Provider Ran Successfully ---')\n"
            "    return 0\n"
        )
        return proj_dir

    return _make_project


def test_run_packaged_provider(
    tmp_path: Path,
    private_key_pem: bytes,
    public_key_pem: bytes,
    make_sample_project: Callable[[Path], Path],
) -> None:
    """
    Tests a full end-to-end flow: package a provider and run the resulting binary.
    """
    runner = CliRunner()

    with runner.isolated_filesystem(temp_dir=tmp_path) as td_str:
        td = Path(td_str)
        make_sample_project(td)

        output_path = td / "dist/terraform-provider-myprovider"
        keys_dir = td / "keys"
        keys_dir.mkdir()

        private_key_path = keys_dir / "provider-private.key"
        public_key_path = keys_dir / "provider-public.key"

        pyproject_toml_content = f"""
[tool.pyvider]
name = "myprovider"
output_path = "{output_path}"
entry_point = "my_sample_pkg.main:serve"

[tool.pyvider.build]
dependencies = ["./sample_project"]

[tool.pyvider.signing]
private_key_path = "{private_key_path.relative_to(td)}"
public_key_path = "{public_key_path.relative_to(td)}"
"""
        (td / "pyproject.toml").write_text(pyproject_toml_content)
        private_key_path.write_bytes(private_key_pem)
        public_key_path.write_bytes(public_key_pem)

        package_result = runner.invoke(cli, ["package"])
        assert package_result.exit_code == 0, (
            f"Package command failed: {package_result.output}"
        )
        assert (
            "✅ Go-based cryptographic verification successful."
            in package_result.output
        )
        assert output_path.exists()

        output_path.chmod(output_path.stat().st_mode | stat.S_IEXEC)

        run_env = os.environ.copy()
        run_env["PYVIDER_LOG_LEVEL"] = "DEBUG"
        run_env["PSPF_INTERACTIVE"] = "false"

        result = subprocess.run(
            [str(output_path)], capture_output=True, text=True, env=run_env, check=False
        )

        assert result.returncode == 0, (
            f"Packaged provider execution failed. Stderr: {result.stderr}"
        )
        assert "--- My Provider Ran Successfully ---" in result.stdout
