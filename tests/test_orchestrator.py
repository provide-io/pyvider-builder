"""Tests for the BuildOrchestrator class."""

import asyncio
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from pyvider.builder.packaging.orchestrator import BuildOrchestrator
from pyvider.schema import PvsSchema, s_resource


@pytest.fixture
def mock_provider_project(tmp_path: Path) -> Path:
    """Creates a mock provider project structure for testing dynamic imports."""
    project_dir = tmp_path / "myprovider"
    src_dir = project_dir / "src"
    pkg_dir = src_dir / "my_pkg"
    pkg_dir.mkdir(parents=True)

    (src_dir / "__init__.py").touch()
    (pkg_dir / "__init__.py").touch()

    (pkg_dir / "main.py").write_text(
        """
from pyvider.schema import PvsSchema, s_resource

async def get_schema_func():
    # Add the required 'version' argument
    return PvsSchema(version=1, block=s_resource({}))
"""
    )
    return project_dir


@pytest.mark.asyncio
async def test_extract_schema(mock_provider_project: Path):
    """
    Tests that the BuildOrchestrator can correctly import a module
    from a given path and execute an async entry point to get a schema.
    """
    orchestrator = BuildOrchestrator(
        launcher_bin_path="/fake/launcher",
        package_integrity_key_path="/fake/private.key",
        public_key_path="/fake/public.key",
        output_pspf_path="/fake/dist/provider",
        build_config={},
        manifest_dir=mock_provider_project,
        entry_point="my_pkg.main:get_schema_func",
        python_version=None,
    )

    schema = await orchestrator.extract_schema()

    assert isinstance(schema, PvsSchema)
