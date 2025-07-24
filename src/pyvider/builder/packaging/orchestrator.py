"""Core logic for building PSPF packages by orchestrating the Go packager CLI."""

from collections.abc import Callable, Iterable
import fnmatch
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
from typing import Any
import sys
import importlib

from pyvider.telemetry import logger

from ..compiler import ensure_go_binary
from ..exceptions import BuildError
from pyvider.schema import PvsSchema


def create_ignore_func(
    root: Path, patterns: list[str]
) -> Callable[[str, list[str]], Iterable[str]]:
    """Creates a function suitable for shutil.copytree's ignore argument."""

    def ignore(dir_path_str: str, names: list[str]) -> Iterable[str]:
        dir_path = Path(dir_path_str)
        ignored_names = set()
        for name in names:
            rel_path_str = str((dir_path / name).relative_to(root))
            for pattern in patterns:
                if fnmatch.fnmatch(rel_path_str, pattern) or fnmatch.fnmatch(
                    name, pattern
                ):
                    ignored_names.add(name)
                    break
        return ignored_names

    return ignore


class BuildOrchestrator:
    DEFAULT_PYTHON_VERSION = "3.13"

    def __init__(
        self,
        launcher_bin_path: str,
        package_integrity_key_path: str,
        public_key_path: str,
        output_pspf_path: str,
        build_config: dict[str, Any],
        manifest_dir: Path,
        entry_point: str,
        python_version: str | None = None,
    ) -> None:
        self.launcher_bin_path = launcher_bin_path
        self.package_integrity_key_path = package_integrity_key_path
        self.public_key_path = public_key_path
        self.output_pspf_path = output_pspf_path
        self.entry_point = entry_point
        self.packager_executable = str(ensure_go_binary("pspf-packager"))
        self.build_config = build_config
        self.manifest_dir = manifest_dir
        self.python_version = python_version or self.DEFAULT_PYTHON_VERSION

    async def extract_schema(self) -> PvsSchema:
        """Extracts the provider schema."""
        src_path = self.manifest_dir / "src"
        if src_path.exists():
            sys.path.insert(0, str(src_path))
        try:
            module_name, _, func_name = self.entry_point.rpartition(":")
            module = importlib.import_module(module_name)
            schema_func = getattr(module, func_name)
            return await schema_func()
        finally:
            if src_path.exists():
                sys.path.pop(0)

    def _run_subprocess(self, command: list[str], cwd: Path | str | None = None) -> str:
        logger.info(f"Running command: {' '.join(command)}")
        result = subprocess.run(
            command, capture_output=True, text=True, cwd=cwd, check=False
        )
        if result.returncode != 0:
            error_message = (
                f"Command failed with exit code {result.returncode}.\n"
                f"  Command: {' '.join(command)}\n"
                f"  Stdout:\n{result.stdout.strip()}\n"
                f"  Stderr:\n{result.stderr.strip()}"
            )
            raise BuildError(error_message)
        if result.stderr:
            logger.debug("Command stderr", output=result.stderr.strip())
        return result.stdout.strip()

    def build_package(self) -> None:
        logger.info("Orchestrator starting manifest-driven build process...")
        output_path = Path(self.output_pspf_path)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        exclude_patterns = self.build_config.get("exclude", [])
        python_executable_path_str = self._run_subprocess(
            ["uv", "python", "find", self.python_version]
        )
        python_install_dir = Path(python_executable_path_str).resolve().parent.parent

        with tempfile.TemporaryDirectory(prefix="pyvider_build_") as temp_dir_str:
            temp_dir = Path(temp_dir_str)
            
            # The Go builder now receives all dependencies directly.
            # We resolve local paths here to make them absolute for the Go builder.
            dependencies = self.build_config.get("dependencies", [])
            resolved_deps = []
            for dep in dependencies:
                dep_path = self.manifest_dir / dep
                if dep_path.exists():
                    resolved_deps.append(str(dep_path.resolve()))
                else:
                    resolved_deps.append(dep)

            config_data = {"entry_point": self.entry_point}
            config_json_path = temp_dir / "config.json"
            config_json_path.write_text(json.dumps(config_data))

            build_cmd_args = [
                self.packager_executable, "build",
                "--launcher-bin", self.launcher_bin_path,
                "--package-key", self.package_integrity_key_path,
                "--public-key", self.public_key_path,
                "--out", self.output_pspf_path,
                "--python-install-dir", str(python_install_dir),
                "--config", str(config_json_path),
            ]
            
            for dep in resolved_deps:
                build_cmd_args.extend(["--dependency", dep])

            for pattern in exclude_patterns:
                build_cmd_args.extend(["--exclude", pattern])
            
            # The Go builder now handles all wheel creation, so we run from a neutral temp dir.
            self._run_subprocess(build_cmd_args, cwd=temp_dir)
