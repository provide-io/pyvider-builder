import struct
import zlib
from typing import Self

from attrs import define, field

# Canonical PSPF v0.1 Specification Constants
PSPF_VERSION_NUMBER: int = 0x0003
PSPF_RESERVED_FIELD: int = 0x0000
PSPF_INTERNAL_FOOTER_MAGIC_NUMBER: int = 0x30505350  # '0PSP'
PSPF_EOF_MAGIC_STRING: bytes = b"!PSPF\x00\x00\x00"

# Format for 12 uint64, 2 uint16, 2 uint32
FOOTER_STRUCT_FORMAT = "<QQQQQQQQQQQQHHII"
FOOTER_SIZE = struct.calcsize(FOOTER_STRUCT_FORMAT)

if FOOTER_SIZE != 108:
    raise AssertionError(
        f"Calculated PSPF footer size is {FOOTER_SIZE}, expected 108."
    )


@define(frozen=True, slots=True)
class PspfFooter:
    uv_binary_offset: int
    uv_binary_size: int
    python_install_tgz_offset: int
    python_install_tgz_size: int
    metadata_tgz_offset: int
    metadata_tgz_size: int
    payload_tgz_offset: int
    payload_tgz_size: int
    package_signature_offset: int
    package_signature_size: int
    public_key_pem_offset: int
    public_key_pem_size: int
    pspf_version: int = field(default=PSPF_VERSION_NUMBER)
    reserved: int = field(default=PSPF_RESERVED_FIELD)
    footer_struct_checksum: int = field(init=False)
    internal_footer_magic: int = field(default=PSPF_INTERNAL_FOOTER_MAGIC_NUMBER)

    def __attrs_post_init__(self) -> None:
        data_to_checksum = struct.pack(
            FOOTER_STRUCT_FORMAT,
            self.uv_binary_offset,
            self.uv_binary_size,
            self.python_install_tgz_offset,
            self.python_install_tgz_size,
            self.metadata_tgz_offset,
            self.metadata_tgz_size,
            self.payload_tgz_offset,
            self.payload_tgz_size,
            self.package_signature_offset,
            self.package_signature_size,
            self.public_key_pem_offset,
            self.public_key_pem_size,
            self.pspf_version,
            self.reserved,
            0,  # Checksum field is 0 for calculation
            self.internal_footer_magic,
        )
        calculated_checksum = zlib.crc32(data_to_checksum) & 0xFFFFFFFF
        object.__setattr__(self, "footer_struct_checksum", calculated_checksum)

    def pack(self) -> bytes:
        return struct.pack(
            FOOTER_STRUCT_FORMAT,
            self.uv_binary_offset,
            self.uv_binary_size,
            self.python_install_tgz_offset,
            self.python_install_tgz_size,
            self.metadata_tgz_offset,
            self.metadata_tgz_size,
            self.payload_tgz_offset,
            self.payload_tgz_size,
            self.package_signature_offset,
            self.package_signature_size,
            self.public_key_pem_offset,
            self.public_key_pem_size,
            self.pspf_version,
            self.reserved,
            self.footer_struct_checksum,
            self.internal_footer_magic,
        )

    @classmethod
    def unpack(cls, buffer: bytes) -> Self:
        if len(buffer) != FOOTER_SIZE:
            raise ValueError(f"Buffer size {len(buffer)} != {FOOTER_SIZE}")

        unpacked = struct.unpack(FOOTER_STRUCT_FORMAT, buffer)

        footer_instance = cls(
            uv_binary_offset=unpacked[0],
            uv_binary_size=unpacked[1],
            python_install_tgz_offset=unpacked[2],
            python_install_tgz_size=unpacked[3],
            metadata_tgz_offset=unpacked[4],
            metadata_tgz_size=unpacked[5],
            payload_tgz_offset=unpacked[6],
            payload_tgz_size=unpacked[7],
            package_signature_offset=unpacked[8],
            package_signature_size=unpacked[9],
            public_key_pem_offset=unpacked[10],
            public_key_pem_size=unpacked[11],
            pspf_version=unpacked[12],
            reserved=unpacked[13],
            internal_footer_magic=unpacked[15],
        )

        read_checksum_from_buffer = unpacked[14]
        if footer_instance.footer_struct_checksum != read_checksum_from_buffer:
            raise ValueError("Footer checksum mismatch.")

        if footer_instance.internal_footer_magic != PSPF_INTERNAL_FOOTER_MAGIC_NUMBER:
            raise ValueError("Invalid InternalFooterMagic.")
        if footer_instance.pspf_version != PSPF_VERSION_NUMBER:
            raise ValueError("Unexpected PSPF version.")

        return footer_instance


PSPF_MAGIC_NUMBER = PSPF_INTERNAL_FOOTER_MAGIC_NUMBER
PSPF_EOF_MAGIC = PSPF_EOF_MAGIC_STRING
PSPF_VERSION = PSPF_VERSION_NUMBER
