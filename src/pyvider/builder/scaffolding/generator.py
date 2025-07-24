"""Logic for scaffolding new provider projects and components."""

from pathlib import Path

import jinja2

_TEMPLATE_DIR = Path(__file__).parent / "templates"


def _get_template_env() -> jinja2.Environment:
    return jinja2.Environment(
        loader=jinja2.FileSystemLoader(_TEMPLATE_DIR),
        trim_blocks=True,
        lstrip_blocks=True,
    )


def _find_project_root(start_path: Path | None = None) -> Path | None:
    """Finds the root of the current provider project."""
    current = (start_path or Path.cwd()).resolve()
    while True:
        pyproject_path = current / "pyproject.toml"
        if pyproject_path.exists() and (current / "src").is_dir():
            return current
        if current.parent == current:
            return None
        current = current.parent


def scaffold_new_provider(provider_name: str, path: str) -> Path:
    """Scaffolds a new Pyvider provider project structure."""
    project_dir = Path(path).resolve() / f"terraform-provider-{provider_name}"
    if project_dir.exists():
        raise FileExistsError(f"Directory already exists: {project_dir}")

    src_root = project_dir / "src"
    provider_src_dir = src_root / provider_name
    resources_dir = provider_src_dir / "components" / "resources"
    data_sources_dir = provider_src_dir / "components" / "data_sources"

    resources_dir.mkdir(parents=True, exist_ok=True)
    data_sources_dir.mkdir(parents=True, exist_ok=True)

    (provider_src_dir / "components" / "__init__.py").touch(exist_ok=True)
    (resources_dir / "__init__.py").touch(exist_ok=True)
    (data_sources_dir / "__init__.py").touch(exist_ok=True)

    env = _get_template_env()
    pyproject_template = env.get_template("pyproject.toml.j2")
    pyproject_content = pyproject_template.render(provider_name=provider_name)
    (project_dir / "pyproject.toml").write_text(pyproject_content)

    main_py_content = f'"""Main entry point for the {provider_name} provider."""\nfrom pyvider.provider_core import setup_provider\n\ndef serve():\n    setup_provider()\n'
    (provider_src_dir / "main.py").write_text(main_py_content)

    gitignore_content = "# Python\n__pycache__/\n*.py[cod]\n*$py.class\n*.egg-info/\n.env\n.venv\ndist/\nbuild/\n*.egg\n\n# Terraform\n.terraform/\n.terraform.lock.hcl\n*.tfstate\n*.tfstate.backup\ncrash.log\n"
    (project_dir / ".gitignore").write_text(gitignore_content)
    return project_dir


def scaffold_add_component(
    component_type: str, component_name: str, start_dir: Path | None = None
) -> Path:
    """Adds a new component to a Pyvider project."""
    project_root = _find_project_root(start_dir)
    if not project_root:
        raise FileNotFoundError(
            "Could not find Pyvider project root (pyproject.toml and src/)."
        )

    src_dir_items = [
        d
        for d in (project_root / "src").iterdir()
        if d.is_dir() and not d.name.startswith("_")
    ]
    if not src_dir_items:
        raise FileNotFoundError(
            "Could not determine the provider source directory under src/."
        )
    provider_name = src_dir_items[0].name

    base_path = project_root / "src" / provider_name / "components"
    target_dir = base_path / f"{component_type}s"
    template_name = f"{component_type}/{component_type}.py.j2"

    target_dir.mkdir(parents=True, exist_ok=True)
    output_file = target_dir / f"{component_name.lower()}.py"

    if output_file.exists():
        raise FileExistsError(f"Component file already exists: {output_file}")

    env = _get_template_env()
    template = env.get_template(template_name)

    component_class_name = "".join(
        word.capitalize() for word in component_name.split("_")
    )

    rendered_content = template.render(
        component_name=component_name,
        component_class_name=component_class_name,
        provider_name=provider_name,
    )
    output_file.write_text(rendered_content)
    return output_file
