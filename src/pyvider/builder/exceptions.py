class BuildError(Exception):
    pass


class SigningError(BuildError):
    pass


class VerificationError(Exception):
    pass


class InvalidFooterError(VerificationError):
    pass


class SignatureVerificationError(VerificationError):
    pass
