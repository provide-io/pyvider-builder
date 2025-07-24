# pyvider/src/pyvider/builder/__init__.py
"""
This package contains the core logic for building Pyvider providers into
distributable artifacts, such as the Progressive Secure Package Format (PSPF).
"""

from .models import (
    PSPF_EOF_MAGIC,
    PSPF_VERSION,
    PspfFooter,
)
from .packaging.orchestrator import BuildOrchestrator

# NOTE: The verifier is NOT imported here to avoid an import cycle.
# The `verify` command in the CLI should import it directly.

__all__ = [
    "PSPF_EOF_MAGIC",
    "PSPF_VERSION",
    "BuildOrchestrator",
    "PspfFooter",
]
