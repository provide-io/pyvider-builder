# CLI Reference (`pyvbuild`)

The `pyvbuild` command is the main entry point for the toolchain.

## `pyvbuild package`

Packages the provider into a PSPF binary based on `pyproject.toml`.

**Usage:**
`pyvbuild package [OPTIONS]`

**Options:**
- `--manifest PATH`: Path to the `pyproject.toml` file. [default: `pyproject.toml`]
- `--out PATH`: Override the `output_path` from the manifest.
- `--private-key-path PATH`: Override the private key path.
- `--public-key-path PATH`: Override the public key path.

## `pyvbuild keygen`

Generates a new RSA key pair for signing.

**Usage:**
`pyvbuild keygen [OPTIONS]`

**Options:**
- `--out-dir PATH`: Directory to save the key pair. [default: `keys`]

## `pyvbuild verify`

Verifies the cryptographic integrity of a PSPF package.

**Usage:**
`pyvbuild verify [PACKAGE_FILE] [OPTIONS]`

**Options:**
- `--public-key-path PATH`: Path to the public key for verification.

## `pyvbuild clean`

Removes cached Go binaries compiled by `pyvider-builder`.

**Usage:**
`pyvbuild clean`
