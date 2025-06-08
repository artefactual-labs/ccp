import re
from typing import TypedDict
from unittest import mock

import pytest

from worker.fpr.models import FPCommand
from worker.utils.executeOrRunSubProcess import executeOrRun


@pytest.fixture
def mock_external_tools():
    """Mock subprocess calls to return expected output for external tools.

    Note: this is unique to the CCP fork, upstream Archivematica expects the
    external tools to be installed and available in the system PATH.
    These tests still validate command execution flow and output parsing logic.
    """
    import os
    import subprocess

    # Expected outputs for each command type
    mock_outputs = {
        "7z": 'program="7z"; version="p7zip Version 16.02"',
        "convert": 'program="convert"; version="Version: ImageMagick 7.1.0-4"',
        "ffmpeg": 'program="ffmpeg"; version="ffmpeg version 4.4.2"',
        "ps2pdf": 'program="ps2pdf"; program="Ghostscript"; version="9.56.1"',
        "Ghostscript": 'program="Ghostscript"; version="9.56.1"',
        "inkscape": 'program="inkscape"; version="Inkscape 1.2"',
        "unrar-nonfree": 'program="unrar-nonfree"; version="UNRAR 6.1.7"',
        "readpst": 'program="readpst"; version="ReadPST / LibPST v0.6.76"',
    }

    original_popen = subprocess.Popen

    def mock_popen(*args, **kwargs):
        command = args[0] if args else kwargs.get("args", [])

        # Check if this is a temporary script execution
        if isinstance(command, list) and len(command) > 0:
            script_path = command[0]
            if os.path.exists(script_path):
                try:
                    with open(script_path) as f:
                        script_content = f.read()

                    # Check which tool is being called in the script
                    for tool, output in mock_outputs.items():
                        if tool in script_content:
                            mock_process = mock.Mock()
                            mock_process.returncode = 0
                            mock_process.communicate.return_value = (
                                output.encode(),
                                b"",
                            )
                            return mock_process

                except OSError:
                    pass

        # Fall back to original Popen for other commands
        return original_popen(*args, **kwargs)

    with mock.patch("subprocess.Popen", side_effect=mock_popen):
        yield


class QueryFilters(TypedDict):
    command_usage: str
    description: str


class EventDetailResult(TypedDict):
    programs: list[str]
    version: str


@pytest.mark.django_db
@pytest.mark.parametrize(
    "expected_programs,expected_version_pattern,cmd,filters",
    [
        (
            ["7z"],
            # The event detail command extracts different lines depending on the 7z version.
            # Older versions report the version on a line starting with "p7zip Version"
            # while more recent versions use a line starting with "7-Zip".
            r"(^p7zip Version|^7-Zip)",
            'echo program=\\"7z\\"\\; version=\\"`7z | awk \'NR==3 && /^p7zip Version/ {print; exit} NR==2 {line2=$0} NR==3 {print line2 $0}\'`\\"',
            {
                "command_usage": "event_detail",
                "description": "Get event detail text for 7z extraction",
            },
        ),
        (
            ["convert"],
            "^Version: ImageMagick",
            'echo program=\\"convert\\"\\; version=\\"`convert -version | grep Version:`\\"',
            {
                "command_usage": "event_detail",
                "description": "convert event detail",
            },
        ),
        (
            ["ffmpeg"],
            r"^ffmpeg version",
            'echo program=\\"ffmpeg\\"\\; version=\\"`ffmpeg 2>&1 | grep --ignore-case "FFmpeg version"`\\"',
            {
                "command_usage": "event_detail",
                "description": "Get event detail text for ffmpeg extraction",
            },
        ),
        (
            ["ps2pdf", "Ghostscript"],
            r"^\d+\.\d+\.\d+",
            'echo program=\\"ps2pdf\\"\\; program=\\"Ghostscript\\"\\; version=\\"`gs --version`\\" ',
            {
                "command_usage": "event_detail",
                "description": "ps2pdf event detail",
            },
        ),
        (
            ["Ghostscript"],
            r"^\d+\.\d+\.\d+",
            'echo program=\\"Ghostscript\\"\\; version=\\"`gs --version`\\" ',
            {
                "command_usage": "event_detail",
                "description": "Ghostscript event detail",
            },
        ),
        (
            ["inkscape"],
            r"^Inkscape",
            'echo program=\\"inkscape\\"\\; version=\\"`inkscape -V`\\" ',
            {
                "command_usage": "event_detail",
                "description": "inkscape event detail",
            },
        ),
        pytest.param(
            ["unrar-nonfree"],
            "^UNRAR",
            'echo program=\\"unrar-nonfree\\"\\; version=\\"`unrar-nonfree | grep \'UNRAR\'`\\"',
            {
                "command_usage": "event_detail",
                "description": "Get event detail text for unrar extraction",
            },
            marks=pytest.mark.skip(
                reason="Skipping because unrar-nonfree is not installed by default in Archivematica"
            ),
        ),
        (
            ["readpst"],
            r"^ReadPST / LibPST",
            'echo program=\\"readpst\\"\\; version=\\"`readpst -V`\\"',
            {
                "command_usage": "event_detail",
                "description": "readpst event detail",
            },
        ),
    ],
    ids=[
        "7z",
        "convert",
        "ffmpeg",
        "ps2pdf",
        "Ghostscript",
        "inkscape",
        "unrar-nonfree",
        "readpst",
    ],
)
def test_event_detail_command_returns_tool_version(
    mock_external_tools,
    expected_programs: list[str],
    expected_version_pattern: str,
    cmd: str,
    filters: QueryFilters,
) -> None:
    command, _ = FPCommand.active.get_or_create(
        script_type="bashScript", command=cmd, **filters
    )

    _, output, _ = executeOrRun(command.script_type, command.command)

    result: EventDetailResult = {"programs": [], "version": ""}

    for match in re.finditer(
        r'program="(?P<program>.*?)";|version="(?P<version>.*?)"', output
    ):
        program = match.group("program")
        version = match.group("version")
        if program is not None:
            result["programs"].append(program)
        if version is not None:
            result["version"] = version

    assert result["programs"] == expected_programs
    assert re.search(expected_version_pattern, result["version"]) is not None
