"""
Centralized cryptographic operations for the Pyvider builder.
"""

from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding, rsa

from .exceptions import SigningError


def generate_keys() -> tuple[rsa.RSAPrivateKey, rsa.RSAPublicKey]:
    """Generates a new 4096-bit RSA key pair."""
    private_key = rsa.generate_private_key(public_exponent=65537, key_size=4096)
    return private_key, private_key.public_key()


def sign_payload_hash(payload_hash: bytes, private_key: rsa.RSAPrivateKey) -> bytes:
    """Signs a 32-byte hash using RSA-PSS, matching the Go implementation."""
    if not isinstance(payload_hash, bytes) or len(payload_hash) != 32:
        raise SigningError("Payload hash must be a 32-byte SHA-256 hash.")

    return private_key.sign(
        payload_hash,
        padding.PSS(
            mgf=padding.MGF1(hashes.SHA256()), salt_length=hashes.SHA256.digest_size
        ),
        hashes.SHA256(),
    )
