"""
TDD tests for the BuildOrchestrator to ensure it correctly passes dependencies
to the Go builder CLI.
"""

from pathlib import Path
from unittest.mock import patch, call

from pyvider.builder.packaging.orchestrator import BuildOrchestrator


def test_orchestrator_passes_dependencies_as_flags(tmp_path: Path):
    """
    TDD Contract: Verifies that the BuildOrchestrator constructs the correct
    `--dependency` flags for the Go builder subprocess call.
    """
    manifest_dir = tmp_path
    (manifest_dir / "src" / "local_pkg").mkdir(parents=True)

    build_config = {
        "dependencies": [
            "attrs>=23.1.0",
            "./src/local_pkg",
        ]
    }

    output_path = tmp_path / "dist" / "provider"

    with patch("pyvider.builder.packaging.orchestrator.BuildOrchestrator._run_subprocess") as mock_run:
        # Configure the mock to return a dummy path for the first call (uv python find)
        mock_run.return_value = "/path/to/python/install/bin/python"

        orchestrator = BuildOrchestrator(
            launcher_bin_path="/fake/launcher",
            package_integrity_key_path=str(tmp_path / "private.key"),
            public_key_path=str(tmp_path / "public.key"),
            output_pspf_path=str(output_path),
            build_config=build_config,
            manifest_dir=manifest_dir,
            entry_point="main:serve",
            python_version="3.13",
        )

        orchestrator.build_package()

        # Assert that the mock was called twice
        assert mock_run.call_count == 2

        # Assert the first call was to find python
        first_call_args = mock_run.call_args_list[0].args[0]
        assert first_call_args == ["uv", "python", "find", "3.13"]

        # Assert the second call was to the Go packager with the correct args
        second_call_args = mock_run.call_args_list[1].args[0]
        assert second_call_args[0].endswith("pspf-packager")
        assert "--dependency" in second_call_args
        assert "attrs>=23.1.0" in second_call_args
        assert str((manifest_dir / "src/local_pkg").resolve()) in second_call_args
