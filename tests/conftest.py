"""Pytest fixtures for the entire pyvider-builder test suite."""

import os
import subprocess
import sys
from typing import Any, Generator

from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
import pytest

from pyvider.builder.crypto import generate_keys


@pytest.fixture(scope="session", autouse=True)
def auto_clean_pspf_cache() -> Generator[None, None, None]:
    """
    A session-scoped, auto-use fixture that runs `pyvbuild clean` once
    before any tests are run.

    This uses a direct subprocess call and sets the NO_COVERAGE environment
    variable. This ensures that the setup command is fully isolated from
    pytest-cov instrumentation, preventing it from generating a conflicting
    coverage data file that would otherwise cause a DataError during the
    final report combination phase.
    """
    py_executable = sys.executable
    
    # Create a copy of the current environment and add NO_COVERAGE to it.
    env = os.environ.copy()
    env["NO_COVERAGE"] = "1"

    result = subprocess.run(
        [py_executable, "-m", "pyvider.builder.cli", "clean"],
        capture_output=True,
        text=True,
        check=False,
        env=env,  # Pass the modified environment to the subprocess.
    )
    assert (
        result.returncode == 0
    ), f"Failed to clean PSPF cache before test session: {result.stderr}"
    yield


@pytest.fixture(scope="session")
def key_pair() -> tuple[rsa.RSAPrivateKey, rsa.RSAPublicKey]:
    """Generates a single RSA key pair for the entire test session."""
    return generate_keys()


@pytest.fixture(scope="session")
def private_key(key_pair: tuple[rsa.RSAPrivateKey, rsa.RSAPublicKey]) -> rsa.RSAPrivateKey:
    """Returns the private key object from the session-scoped key pair."""
    return key_pair[0]


@pytest.fixture(scope="session")
def public_key(key_pair: tuple[rsa.RSAPrivateKey, rsa.RSAPublicKey]) -> rsa.RSAPublicKey:
    """Returns the public key object from the session-scoped key pair."""
    return key_pair[1]


@pytest.fixture(scope="session")
def public_key_pem(public_key: rsa.RSAPublicKey) -> bytes:
    """Provides the public key serialized in PEM format."""
    return public_key.public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo,
    )


@pytest.fixture(scope="session")
def private_key_pem(private_key: rsa.RSAPrivateKey) -> bytes:
    """Provides the private key serialized in PEM format."""
    return private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )
