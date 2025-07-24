"""
TDD test to ensure the Python PspfFooter model is consistent with the
canonical 108-byte Go specification.
"""
import struct

from pyvider.builder.models import FOOTER_SIZE, FOOTER_STRUCT_FORMAT


def test_python_footer_model_matches_go_specification():
    """
    TDD Contract: Verifies that the Python attrs model for the PSPF footer
    has the exact size and struct format string required by the canonical
    Go implementation.
    """
    # The canonical Go implementation uses a 108-byte footer.
    EXPECTED_FOOTER_SIZE = 108
    # The format is 12 uint64s, 2 uint16s, and 2 uint32s, little-endian.
    EXPECTED_STRUCT_FORMAT = "<QQQQQQQQQQQQHHII"

    assert FOOTER_SIZE == EXPECTED_FOOTER_SIZE, \
        f"Python footer model size is {FOOTER_SIZE}, but must be {EXPECTED_FOOTER_SIZE}."

    assert FOOTER_STRUCT_FORMAT == EXPECTED_STRUCT_FORMAT, \
        f"Python footer struct format is '{FOOTER_STRUCT_FORMAT}', but must be '{EXPECTED_STRUCT_FORMAT}'."

    # Also verify the calculated size from the format string
    assert struct.calcsize(FOOTER_STRUCT_FORMAT) == EXPECTED_FOOTER_SIZE, \
        "The struct format string itself does not calculate to the expected size."
