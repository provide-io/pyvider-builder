"""Tests for the signing module."""

import pytest
from cryptography.hazmat.primitives.asymmetric import rsa

# FIX: Import from the new canonical crypto module.
from pyvider.builder.crypto import sign_payload_hash
from pyvider.builder.exceptions import SigningError


def test_sign_payload_hash_invalid_input(private_key: rsa.RSAPrivateKey) -> None:
    """Tests that sign_payload_hash raises SigningError for invalid hash."""
    with pytest.raises(
        SigningError, match="Payload hash must be a 32-byte SHA-256 hash."
    ):
        sign_payload_hash(b"not a 32-byte hash", private_key)
