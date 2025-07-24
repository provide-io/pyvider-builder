# Configuration (`pyproject.toml`)

All build configuration resides in the `[tool.pyvider]` table of your `pyproject.toml` file.

## `[tool.pyvider]` Table

| Key              | Required | Description                                                  |
| :--------------- | :------- | :----------------------------------------------------------- |
| `output_path`    | Yes      | Path for the final provider binary.                          |
| `entry_point`    | Yes      | The Python entry point for the provider (e.g., `pkg.main:serve`). |
| `python_version` | No       | The Python version to embed (e.g., "3.13"). Defaults to "3.13". |

## `[tool.pyvider.build]` Table

| Key            | Required | Description                                                  |
| :------------- | :------- | :----------------------------------------------------------- |
| `dependencies` | Yes      | A list of all Python dependencies. This includes local paths to your source code (e.g., `"./src/myprovider"`) and PyPI specifiers (e.g., `"attrs>=23.1.0"`). |
| `exclude`      | No       | A list of glob patterns to exclude from the package archives. |

## `[tool.pyvider.signing]` Table

| Key               | Required | Description                                      |
| :---------------- | :------- | :----------------------------------------------- |
| `private_key_path`| Yes      | Path to the private key for signing the package. |
| `public_key_path` | Yes      | Path to the public key to embed in the package.  |
