"""
Main entry point for the {{ provider_name }} Terraform provider.
"""
from pyvider.provider_bridge import Provider


def serve():
    """Serve the provider using Pyvider framework."""
    provider = Provider(provider_name="{{ provider_name }}")
    provider.serve()


if __name__ == "__main__":
    serve()
