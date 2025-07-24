[project]
name = "terraform-provider-{{ provider_name }}"
version = "0.1.0"
description = "A Terraform provider for {{ provider_name }}, built with Pyvider."
readme = "README.md"
requires-python = ">=3.13"
dependencies = [
    "pyvider>=0.0.13",
]

[project.entry-points."terraform.providers"]
{{ provider_name }} = "{{ provider_name }}.main:serve"

[build-system]
requires = ["setuptools>=61.0", "wheel", "pyvider-builder>=0.1.0"]
build-backend = "setuptools.build_meta"

[tool.pyvider-builder]
provider_name = "{{ provider_name }}"
entry_point = "{{ provider_name }}.main:serve"

[tool.setuptools.packages.find]
where = ["src"]

[tool.setuptools.package-data]
"*" = ["*.pyi", "py.typed"]
