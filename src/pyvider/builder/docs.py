import importlib
import sys
from pathlib import Path

from jinja2 import Environment, FileSystemLoader
from pyvider.schema import PvsSchema as ProviderSchema
from pyvider.cty import CtyObject
from pyvider.schema.types.attribute import PvsAttribute


def render_documentation(
    provider_schema: ProviderSchema,
    templates_dir: Path,
    output_dir: Path,
    docs: dict[str, str],
) -> None:
    """Renders the documentation for the given provider schema."""
    env = Environment(loader=FileSystemLoader(templates_dir))

    for name, attribute in provider_schema.block.attributes.items():
        if isinstance(attribute.type, CtyObject):
            if attribute.description == "resource":
                template = env.get_template("resource.md.j2")
                rendered = template.render(
                    resource_name=name,
                    resource=attribute.object_type,
                    docs=docs,
                )
                output_file = output_dir / f"{name}.md"
                output_file.write_text(rendered)
            elif attribute.description == "data source":
                template = env.get_template("data_source.md.j2")
                rendered = template.render(
                    data_source_name=name,
                    data_source=attribute.object_type,
                    docs=docs,
                )
                output_file = output_dir / f"{name}.md"
                output_file.write_text(rendered)
        elif isinstance(attribute, PvsAttribute) and callable(
            attribute.type
        ):  # It's a function
            template = env.get_template("function.md.j2")
            rendered = template.render(
                function_name=name, function=attribute, docs=docs
            )
            output_file = output_dir / f"{name}.md"
            output_file.write_text(rendered)


def generate_docs(provider_dir: Path, schema: ProviderSchema) -> None:
    """Generates documentation for the provider in the given directory."""
    print(f"provider_dir: {provider_dir}")
    print(f"schema: {schema}")
    # Find the templates directory
    templates_dir = provider_dir / "docs" / "templates"
    if not templates_dir.is_dir():
        # Use the default templates
        templates_dir = Path(__file__).parent / "templates"

    # Create the output directory
    output_dir = provider_dir / "docs" / "generated"
    output_dir.mkdir(parents=True, exist_ok=True)

    # Read the existing documentation
    docs_dir = provider_dir / "docs"
    docs = {}
    for path in docs_dir.glob("**/*.md"):
        if "generated" not in path.parts:
            key = ".".join(path.relative_to(docs_dir).with_suffix("").parts)
            docs[key] = path.read_text()

    render_documentation(schema, templates_dir, output_dir, docs)
