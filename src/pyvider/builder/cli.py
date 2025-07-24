"""The `pyvbuild` command-line interface."""

import importlib.metadata
from pathlib import Path
import shutil
import subprocess
import tomllib
from typing import Any

import click

from .compiler import _get_cache_dir, ensure_go_binary
from .exceptions import BuildError, InvalidFooterError
from .packaging.orchestrator import BuildOrchestrator
from .packaging.reader import PspfReader

try:
    __version__ = importlib.metadata.version("pyvider-builder")
except importlib.metadata.PackageNotFoundError:
    __version__ = "0.0.0-dev"


@click.group(context_settings=dict(help_option_names=["-h", "--help"]))
@click.version_option(
    __version__,
    "-V",
    "--version",
    prog_name="pyvbuild",
    message="%(prog)s version %(version)s",
)
def cli() -> None:
    """Progressive Secure Packaging Format (PSPF) Build Tool."""
    pass


@cli.command()
@click.option(
    "--out-dir",
    default="keys",
    type=click.Path(file_okay=False, writable=True, resolve_path=True),
    help="Directory to save the PSS RSA key pair.",
)
def keygen(out_dir: str) -> None:
    """Generates a PSS RSA key pair for PSPF package integrity signing."""
    try:
        out_path = Path(out_dir)
        out_path.mkdir(parents=True, exist_ok=True)
        packager_executable = ensure_go_binary("pspf-packager")
        keygen_cmd_args = [
            str(packager_executable),
            "keygen",
            "--out-dir",
            str(out_path),
        ]
        result = subprocess.run(
            keygen_cmd_args, capture_output=True, text=True, check=False
        )

        if result.returncode != 0:
            if "key already exists" in result.stderr:
                click.secho(
                    f"⚠️  Keys already exist. To regenerate, please delete them first.\n{result.stderr.strip()}",
                    fg="yellow",
                )
                return
            else:
                raise subprocess.CalledProcessError(
                    result.returncode, result.args, stderr=result.stderr
                )

        click.secho(
            f"✅ Package integrity key pair generated in '{out_dir}'.", fg="green"
        )
    except (BuildError, subprocess.CalledProcessError) as e:
        stderr_info = (
            f"\n  Stderr: {e.stderr.strip()}"
            if hasattr(e, "stderr") and e.stderr
            else ""
        )
        click.secho(f"❌ Keygen failed: {e}{stderr_info}", fg="red", err=True)
        raise click.Abort() from e


@cli.command("package")
@click.option(
    "--private-key-path",
    type=click.Path(dir_okay=False, resolve_path=True),
    help="Override the private key path from pyproject.toml.",
)
@click.option(
    "--public-key-path",
    type=click.Path(dir_okay=False, resolve_path=True),
    help="Override the public key path from pyproject.toml.",
)
@click.option(
    "--out",
    type=click.Path(resolve_path=True),
    help="Override the output path from pyproject.toml.",
)
@click.option(
    "--manifest",
    "pyproject_toml_path",
    default="pyproject.toml",
    type=click.Path(exists=True, dir_okay=False, resolve_path=True),
    help="Path to the pyproject.toml manifest file.",
)
@click.pass_context
def package_command(
    ctx: click.Context,
    private_key_path: str | None,
    public_key_path: str | None,
    out: str | None,
    pyproject_toml_path: str,
) -> None:
    """Packages the provider and immediately verifies it."""
    click.echo("🚀 Packaging provider...")
    try:
        manifest_path = Path(pyproject_toml_path)
        with manifest_path.open("rb") as f:
            pyproject_data = tomllib.load(f)

        pyvider_conf = pyproject_data.get("tool", {}).get("pyvider", {})
        if not pyvider_conf:
            raise click.UsageError(
                "A [tool.pyvider] section was not found in pyproject.toml."
            )

        build_conf = pyvider_conf.get("build", {})
        signing_conf = pyvider_conf.get("signing", {})
        manifest_dir = manifest_path.parent

        final_out = Path(
            out
            or manifest_dir
            / pyvider_conf.get("output_path", "dist/terraform-provider-pyvider")
        )
        final_key = Path(
            private_key_path
            or manifest_dir
            / signing_conf.get("private_key_path", "keys/provider-private.key")
        )
        final_pub_key = Path(
            public_key_path
            or manifest_dir
            / signing_conf.get("public_key_path", "keys/provider-public.key")
        )
        entry_point = pyvider_conf.get("entry_point")
        python_version = pyvider_conf.get("python_version")

        if not all([final_out, final_key, final_pub_key, entry_point]):
            raise click.UsageError(
                "Missing required configuration. Check paths in [tool.pyvider] or provide them as CLI options."
            )

        if not final_key.exists():
            raise click.UsageError(
                f"Private key not found at '{final_key}'. Please run `pyvbuild keygen` to generate keys."
            )
        if not final_pub_key.exists():
            raise click.UsageError(
                f"Public key not found at '{final_pub_key}'. Please run `pyvbuild keygen` to generate keys."
            )

        launcher_bin_path = ensure_go_binary("pspf-launcher")

        orchestrator = BuildOrchestrator(
            launcher_bin_path=str(launcher_bin_path),
            package_integrity_key_path=str(final_key),
            public_key_path=str(final_pub_key),
            output_pspf_path=str(final_out),
            build_config=build_conf,
            manifest_dir=manifest_dir,
            entry_point=entry_point,
            python_version=python_version,
        )
        orchestrator.build_package()
        click.secho(f"✅ Package built successfully: {final_out}", fg="green")

        click.echo("\n" + "=" * 20 + " Auto-Verification " + "=" * 20)
        ctx.invoke(
            verify_command,
            package_file=str(final_out),
            public_key_path=str(final_pub_key),
        )

    except (BuildError, click.UsageError) as e:
        click.secho(f"❌ Packaging Failed:\n{e}", fg="red", err=True)
        raise click.Abort() from e


@cli.command("verify")
@click.argument(
    "package_file",
    type=click.Path(exists=True, dir_okay=False, resolve_path=True),
    default=None,
    required=False,
)
@click.option(
    "--public-key-path", type=click.Path(exists=True, dir_okay=False, resolve_path=True)
)
def verify_command(
    package_file: str | None, public_key_path: str | None
) -> None:
    """Verifies a PSPF package."""
    manifest_path = Path("pyproject.toml")
    final_package_file = Path(package_file) if package_file else None
    final_public_key = Path(public_key_path) if public_key_path else None

    if not final_package_file or not final_public_key:
        if not manifest_path.exists():
            raise click.UsageError(
                "Cannot find pyproject.toml to determine defaults. Please provide the package file and public key directly."
            )
        with manifest_path.open("rb") as f:
            pyproject_data = tomllib.load(f)

        pyvider_conf = pyproject_data.get("tool", {}).get("pyvider", {})
        manifest_dir = manifest_path.parent

        if not final_package_file:
            final_package_file = manifest_dir / pyvider_conf.get("output_path", "")
        if not final_public_key:
            pub_key_rel_path = pyvider_conf.get("signing", {}).get(
                "public_key_path", "keys/provider-public.key"
            )
            final_public_key = manifest_dir / pub_key_rel_path

    click.echo(f"🔍 Verifying package '{final_package_file}'...")
    try:
        reader = PspfReader(final_package_file)
        click.echo(reader.get_info())

        packager_executable = ensure_go_binary("pspf-packager")
        verify_cmd_args = [
            str(packager_executable),
            "verify",
            str(final_package_file),
            "--public-key",
            str(final_public_key),
        ]
        result = subprocess.run(
            verify_cmd_args, check=True, capture_output=True, text=True
        )
        click.secho("✅ Go-based cryptographic verification successful.", fg="green")
        if result.stderr:
            click.echo(result.stderr)
    except InvalidFooterError as e:
        click.secho(f"❌ Python-based verification failed: {e}", fg="red", err=True)
        raise click.Abort() from e
    except (BuildError, subprocess.CalledProcessError) as e:
        stderr_info = (
            f"\n  Stderr: {e.stderr.strip()}"
            if hasattr(e, "stderr") and e.stderr
            else ""
        )
        click.secho(
            f"❌ Go-based verification failed: {e}{stderr_info}", fg="red", err=True
        )
        raise click.Abort() from e


@cli.command("clean")
def clean_command() -> None:
    """Removes cached Go binaries."""
    click.echo("🧹 Cleaning cached Go binaries...")
    cache_dir = _get_cache_dir()
    bin_dir = cache_dir / "bin"
    if bin_dir.exists():
        shutil.rmtree(bin_dir)
        click.secho(f"✅ Removed cache directory: {bin_dir}", fg="green")
    else:
        click.secho("i️ Cache directory not found, nothing to clean.", fg="yellow")


main = cli
