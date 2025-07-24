# **Progressive Secure Package Format (PSPF) Specification v0.3**

**Status of This Memo:** This document specifies a standards-track protocol for the Pyvider community and requests discussion and suggestions for improvement. Distribution of this memo is unlimited.

### 1. Abstract

This document specifies the Progressive Secure Package Format (PSPF) version 0.3, a file format designed for the secure, self-contained distribution and execution of Python-based applications, specifically tailored for environments like Terraform providers. PSPF v0.3 implements a robust **"verify-then-run"** security model. In this model, the entire package—including an embedded Go launcher, a self-contained Python installation, the `uv` package manager, and all metadata—is cryptographically signed as a single unit. The Go launcher, as the executable entry point, first verifies the integrity and authenticity of the complete package before any part of the Python payload is extracted or executed. This architecture provides a clear and strong security boundary, eliminates external dependencies, and ensures predictable, reproducible runtime environments.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in BCP 14.

### 2. File Format Specification (Version 0.3)

A PSPF v0.3 package MUST be a single, contiguous file.

#### 2.1. File Layout Diagram

```
                            +----------------------------------+
                            |                                  |
                            |  Go Launcher Code                |
                            |                                  |
+-------------------------> +----------------------------------+ <-- UvBinaryOffset
|                           |                                  |
|                           |  Embedded UV Binary              |
|                           |                                  |
|                           +----------------------------------+ <-- PythonInstallTgzOffset
|                           |                                  |
|       Signed Content      |  Embedded Python Install (.tgz)  |
|      (SHA-256 hash of     |                                  |
|       these five blocks   +----------------------------------+ <-- MetadataTgzOffset
|       is signed)          |                                  |
|                           |  Metadata Tarball (metadata.tgz) |
|                           |                                  |
|                           +----------------------------------+ <-- PayloadTgzOffset
|                           |                                  |
|                           |  Payload Tarball (payload.tgz)   |
|                           |                                  |
+-------------------------> +----------------------------------+ <-- PackageSignatureOffset
                            |                                  |
                            |  Signature (RSA-4096 with PSS)   |
                            |                                  |
                            +----------------------------------+ <-- PublicKeyPEMOffset
                            |                                  |
                            |  Embedded Public Key PEM         |
                            |  (Variable Size)                 |
                            |                                  |
                            +----------------------------------+ <-- Footer Offset
                            |                                  |
                            |  PSPF v0.3 Footer (108 bytes)    |
                            |                                  |
                            +----------------------------------+
                            |  EOF Magic String (8 bytes)      |
                            +----------------------------------+
                            |         End of File (EOF)        |
                            +----------------------------------+
```

#### 2.2. PSPF v0.3 Footer Details

The footer is a **108-byte** block located immediately before the final 8-byte EOF Magic String. All multi-byte integer fields MUST be encoded in **little-endian** format.

| Field Name                 | Data Type | Size (Bytes) | Description |
| :------------------------- | :-------- | :----------- | :---------- |
| `UvBinaryOffset`           | `uint64`  | 8            | Absolute byte offset to the start of the UV Binary block. |
| `UvBinarySize`             | `uint64`  | 8            | Size of the UV Binary block in bytes. |
| `PythonInstallTgzOffset`   | `uint64`  | 8            | Absolute byte offset to the start of the Python Install tarball. |
| `PythonInstallTgzSize`     | `uint64`  | 8            | Size of the Python Install tarball in bytes. |
| `MetadataTgzOffset`        | `uint64`  | 8            | Absolute byte offset to the start of the `metadata.tgz` block. |
| `MetadataTgzSize`          | `uint64`  | 8            | Size of the `metadata.tgz` block in bytes. |
| `PayloadTgzOffset`         | `uint64`  | 8            | Absolute byte offset to the start of the `payload.tgz` block. |
| `PayloadTgzSize`           | `uint64`  | 8            | Size of the `payload.tgz` block in bytes. |
| `PackageSignatureOffset`   | `uint64`  | 8            | Absolute byte offset to the start of the Signature block. |
| `PackageSignatureSize`     | `uint64`  | 8            | Size of the Signature block in bytes. |
| `PublicKeyPEMOffset`       | `uint64`  | 8            | Absolute byte offset to the start of the embedded Public Key PEM block. |
| `PublicKeyPEMSize`         | `uint64`  | 8            | Size of the embedded Public Key PEM block in bytes. |
| `PspfVersion`              | `uint16`  | 2            | The version of this specification. MUST be `0x0003`. |
| `Reserved`                 | `uint16`  | 2            | Reserved for future use. MUST be `0`. |
| `FooterStructChecksum`     | `uint32`  | 4            | A CRC32 (IEEE) checksum of all other fields in this footer. |
| `InternalFooterMagic`      | `uint32`  | 4            | ASCII identifier for the footer struct: `PSP0`. Hex: `0x30505350`. |

#### 2.3. EOF Magic String

The file MUST end with the 8-byte magic string `!PSPF\x00\x00\x00`. This serves as a reliable anchor for locating the footer.

### 3. Security Considerations

The security of PSPF v0.3 relies on the "verify-then-run" model. The single digital signature covers all executable code (Launcher, UV, Python) and configuration. Any modification to the package will invalidate the signature, causing the Launcher to terminate before any potentially malicious code is executed.
