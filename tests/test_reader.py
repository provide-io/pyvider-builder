"""Tests for the PSPF reader."""

from pathlib import Path

import pytest

from pyvider.builder.exceptions import InvalidFooterError
from pyvider.builder.packaging.reader import PspfReader


def test_reader_file_not_found() -> None:
    """Tests that the reader raises FileNotFoundError for a non-existent file."""
    with pytest.raises(FileNotFoundError):
        PspfReader(Path("/tmp/non-existent-pspf-file"))


def test_reader_invalid_eof_magic(tmp_path: Path) -> None:
    """Tests that the reader raises InvalidFooterError for bad EOF magic."""
    bad_file = tmp_path / "bad.pspf"
    bad_file.write_bytes(b"this is not a valid file")
    with pytest.raises(InvalidFooterError, match="Invalid PSPF EOF Magic"):
        PspfReader(bad_file)
