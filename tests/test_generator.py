"""Tests for the scaffolding generator."""
from pathlib import Path

import pytest

from pyvider.builder.scaffolding.generator import (
    _find_project_root,
    scaffold_add_component,
    scaffold_new_provider,
)


def test_scaffold_new_provider(tmp_path: Path) -> None:
    """Tests the creation of a new provider project."""
    provider_name = "mytestprovider"
    project_dir = scaffold_new_provider(provider_name, str(tmp_path))

    assert project_dir.name == f"terraform-provider-{provider_name}"
    assert (project_dir / "pyproject.toml").exists()
    assert (project_dir / "src" / provider_name / "main.py").exists()
    assert (
        project_dir / "src" / provider_name / "components" / "resources"
    ).exists()
    assert (
        project_dir / "src" / provider_name / "components" / "data_sources"
    ).exists()


def test_scaffold_new_provider_already_exists(tmp_path: Path) -> None:
    """Tests that scaffolding fails if the directory already exists."""
    provider_name = "mytestprovider"
    project_dir = tmp_path / f"terraform-provider-{provider_name}"
    project_dir.mkdir()

    with pytest.raises(FileExistsError):
        scaffold_new_provider(provider_name, str(tmp_path))


def test_scaffold_add_component(tmp_path: Path) -> None:
    """Tests adding a new component to a provider project."""
    provider_name = "mytestprovider"
    scaffold_new_provider(provider_name, str(tmp_path))
    project_dir = tmp_path / f"terraform-provider-{provider_name}"

    component_path = scaffold_add_component(
        "resource", "my_resource", start_dir=project_dir
    )
    assert component_path.exists()
    assert component_path.name == "my_resource.py"
    assert component_path.parent.name == "resources"

    component_path = scaffold_add_component(
        "data_source", "my_data_source", start_dir=project_dir
    )
    assert component_path.exists()
    assert component_path.name == "my_data_source.py"
    assert component_path.parent.name == "data_sources"


def test_scaffold_add_component_no_project(tmp_path: Path) -> None:
    """Tests that adding a component fails if not in a project."""
    with pytest.raises(FileNotFoundError):
        scaffold_add_component("resource", "my_resource", start_dir=tmp_path)


def test_find_project_root(tmp_path: Path) -> None:
    """Tests the _find_project_root function."""
    project_dir = tmp_path / "myproject"
    src_dir = project_dir / "src"
    src_dir.mkdir(parents=True)
    (project_dir / "pyproject.toml").touch()

    assert _find_project_root(src_dir) == project_dir

    nested_dir = src_dir / "some" / "nested" / "dir"
    nested_dir.mkdir(parents=True)
    assert _find_project_root(nested_dir) == project_dir


def test_find_project_root_not_found(tmp_path: Path) -> None:
    """Tests that _find_project_root returns None when no project is found."""
    assert _find_project_root(tmp_path) is None
