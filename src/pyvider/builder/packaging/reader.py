"""Python-based reader for PSPF package metadata."""

from pathlib import Path

from ..exceptions import InvalidFooterError
from ..models import FOOTER_SIZE, PSPF_EOF_MAGIC, PspfFooter


class PspfReader:
    """Reads and interprets the footer of a PSPF file."""

    def __init__(self, package_path: Path) -> None:
        if not package_path.is_file():
            raise FileNotFoundError(f"Package not found at: {package_path}")
        self.package_path = package_path
        self.footer = self._read_and_verify_footer()

    def _read_and_verify_footer(self) -> PspfFooter:
        """Reads and validates the PSPF footer from the end of the file."""
        with self.package_path.open("rb") as f:
            f.seek(-len(PSPF_EOF_MAGIC), 2)
            eof_magic_bytes = f.read(len(PSPF_EOF_MAGIC))
            if eof_magic_bytes != PSPF_EOF_MAGIC:
                raise InvalidFooterError(
                    f"Invalid PSPF EOF Magic. Found {eof_magic_bytes!r}."
                )

            f.seek(-(FOOTER_SIZE + len(PSPF_EOF_MAGIC)), 2)
            footer_bytes = f.read(FOOTER_SIZE)

        try:
            return PspfFooter.unpack(footer_bytes)
        except ValueError as e:
            raise InvalidFooterError(f"PSPF Footer validation failed: {e}") from e

    def get_info(self) -> str:
        """Returns a human-readable string of the package information."""
        f = self.footer
        return (
            f"PSPF Package Information (parsed by Python):\n"
            f"  PSPF Version: 0x{f.pspf_version:04x}\n"
            f"  UV Binary Size: {f.uv_binary_size} bytes\n"
            f"  Python Install Size: {f.python_install_tgz_size} bytes\n"
            f"  Metadata Size: {f.metadata_tgz_size} bytes\n"
            f"  Payload Size: {f.payload_tgz_size} bytes\n"
            f"  Signature Size: {f.package_signature_size} bytes"
        )
