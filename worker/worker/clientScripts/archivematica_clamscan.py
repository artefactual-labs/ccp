#!/usr/bin/env python
# This file is part of Archivematica.
#
# Copyright 2010-2024 Artefactual Systems Inc. <http://artefactual.com>
#
# Archivematica is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Archivematica is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Archivematica.  If not, see <http://www.gnu.org/licenses/>.
"""Virus scanner compatible with ClamAV.

Settings available:

- ``CLAMAV_CLIENT_BACKEND`` (str)
- ``CLAMAV_SERVER`` (str)
- ``CLAMAV_PASS_BY_STREAM`` (bool)
- ``CLAMAV_CLIENT_TIMEOUT`` (float)
- ``CLAMAV_CLIENT_MAX_FILE_SIZE`` (float)
- ``CLAMAV_CLIENT_MAX_SCAN_SIZE`` (float)
"""

import argparse
import dataclasses
import multiprocessing
import os
import uuid
from typing import List
from typing import Literal
from typing import Optional
from typing import TypedDict
from typing import cast

import django
from clamav_client import get_scanner
from clamav_client.scanner import Scanner
from clamav_client.scanner import ScannerConfig
from django.conf import settings
from django.core.exceptions import ValidationError
from django.db import transaction
from django.utils.functional import LazyObject

django.setup()

from worker.client.job import Job
from worker.main.models import Event
from worker.main.models import File
from worker.utils.custom_handlers import get_script_logger
from worker.utils.databaseFunctions import insertIntoEvents

logger = get_script_logger("archivematica.worker.clamscan")


def concurrent_instances() -> int:
    return multiprocessing.cpu_count()


CLAMD_NAMES = ("clamdscanner", "clamd")
CLAMSCAN_NAMES = ("clamscanner", "clamscan")


EventOutcome = Literal["Pass", "Fail"]


class QueuedEvent(TypedDict):
    fileUUID: str
    eventIdentifierUUID: str
    eventType: str
    eventDateTime: str
    eventDetail: str
    eventOutcome: EventOutcome


EventQueue = List[QueuedEvent]


@dataclasses.dataclass
class Args:
    file_uuid: str
    path: str
    date: str


def file_already_scanned(file_id: uuid.UUID) -> bool:
    qs = Event.objects.filter(file_uuid_id=file_id, event_type="virus check")
    return cast(bool, qs.exists())


def queue_event(
    scanner: Scanner,
    queue: EventQueue,
    file_id: uuid.UUID,
    date: str,
    passed: bool,
) -> None:
    event_detail = ""
    info = scanner.info()  # This is cached.
    event_detail = f'program="{info.name}"; version="{info.version}"; virusDefinitions="{info.virus_definitions}"'
    outcome: EventOutcome = "Pass" if passed else "Fail"
    logger.info("Recording new event for file %s (outcome: %s)", file_id, outcome)
    queue.append(
        {
            "fileUUID": str(file_id),
            "eventIdentifierUUID": str(uuid.uuid4()),
            "eventType": "virus check",
            "eventDateTime": date,
            "eventDetail": event_detail,
            "eventOutcome": outcome,
        }
    )


def get_size(file_id: uuid.UUID, path: str) -> Optional[int]:
    # We're going to see this happening when files are not part of `objects/`.
    try:
        size = File.objects.get(uuid=file_id).size
        return cast(Optional[int], size)
    except (File.DoesNotExist, ValidationError):
        pass
    # Our fallback.
    try:
        return os.path.getsize(path)
    except Exception:
        return None


def valid_max_settings(size: int, max_file_size: float, max_scan_size: float) -> bool:
    max_file_size = max_file_size * 1024 * 1024
    max_scan_size = max_scan_size * 1024 * 1024
    if size > max_file_size:
        logger.info(
            "File will not be scanned. Size %s bytes greater than scanner "
            "max file size %s bytes",
            size,
            max_file_size,
        )
        return False
    elif size > max_scan_size:
        logger.info(
            "File will not be scanned. Size %s bytes greater than scanner "
            "max scan size %s bytes",
            size,
            max_scan_size,
        )
        return False
    return True


def scan_file(
    scanner: Scanner,
    event_queue: EventQueue,
    opts: Args,
) -> int:
    """Scan an individual file.

    Returns 1 to indicate that analyzing the file was impossible.
    Returns 0 if the file can proceed through the process without errors.
    """
    try:
        file_id = uuid.UUID(opts.file_uuid)
    except Exception as err:
        logger.error("File skipped: file_uuid (%s) is not a valid UUID.", err)
        return 1
    if file_already_scanned(file_id):
        logger.info("Virus scan already performed, not running scan again")
        return 0
    passed: Optional[bool] = False
    try:
        size = get_size(file_id, opts.path)
        if size is None:
            passed = None
            logger.error("Getting file size returned: %s", size)
            return 1
        if valid_max_settings(
            size,
            settings.CLAMAV_CLIENT_MAX_FILE_SIZE,
            settings.CLAMAV_CLIENT_MAX_SCAN_SIZE,
        ):
            result = scanner.scan(opts.path)
            passed = True
            state = result.state
            details = result.details
        else:
            passed, state, details = None, None, None
    except Exception:
        logger.error("Unexpected error scanning file %s", opts.path, exc_info=True)
        return 1
    else:
        # record pass or fail, but not None if the file hasn't
        # been scanned, e.g. Max File Size thresholds being too low.
        if passed is not None:
            logger.info("File %s scanned!", opts.path)
            logger.debug("passed=%s state=%s details=%s", passed, state, details)
    finally:
        if passed is not None:
            queue_event(scanner, event_queue, file_id, opts.date, passed)
    return 1 if passed is False else 0


def build_scanner(settings: LazyObject) -> Scanner:
    backend = str(settings.CLAMAV_CLIENT_BACKEND).lower()
    config: ScannerConfig
    if backend in CLAMD_NAMES:
        config = {
            "backend": "clamd",
            "address": str(settings.CLAMAV_SERVER),
            "timeout": int(settings.CLAMAV_CLIENT_TIMEOUT),
            "stream": bool(settings.CLAMAV_PASS_BY_STREAM),
        }
    elif backend in CLAMSCAN_NAMES:
        config = {
            "backend": "clamscan",
            "max_file_size": float(settings.CLAMAV_CLIENT_MAX_FILE_SIZE),
            "max_scan_size": float(settings.CLAMAV_CLIENT_MAX_SCAN_SIZE),
        }
    return get_scanner(config)


def get_parser() -> argparse.ArgumentParser:
    """Return a ``Namespace`` with the parsed arguments."""
    parser = argparse.ArgumentParser()
    parser.add_argument("file_uuid", metavar="fileUUID")
    parser.add_argument("path", metavar="PATH", help="File or directory location")
    parser.add_argument("date", metavar="DATE")
    return parser


def parse_args(parser: argparse.ArgumentParser, job: Job) -> Args:
    namespace = parser.parse_args(job.args[1:])

    return Args(**vars(namespace))


def main(jobs: List[Job]) -> None:
    parser = get_parser()
    event_queue: EventQueue = []
    scanner = build_scanner(settings)
    info = scanner.info()
    logger.info(
        "Using scanner %s (%s - %s)", info.name, info.version, info.virus_definitions
    )

    # TODO: refactor to scan the entire batch of jobs at once, rather than
    # processing one job at a time. This is particularly beneficial when using
    # ClamscanScanner because we only ask ClamAV to load the signatures once.
    for job in jobs:
        with job.JobContext(logger=logger):
            opts = parse_args(parser, job)
            job.set_status(scan_file(scanner, event_queue, opts))
    with transaction.atomic():
        for e in event_queue:
            insertIntoEvents(**e)


def call(jobs: List[Job]) -> None:
    main(jobs)
