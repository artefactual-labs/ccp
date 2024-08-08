#!/usr/bin/env python
# This file is part of Archivematica.
#
# Copyright 2010-2012 Artefactual Systems Inc. <http://artefactual.com>
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
import argparse
import os

import django

django.setup()
from client import metrics

from custom_handlers import get_script_logger
from django.db import transaction

logger = get_script_logger("archivematica.mcp.client.storeAIP")


def store_aip(job, aip_path, sip_uuid, sip_name, sip_type):
    """Stores an AIP or a DIP.

    aip_path = Full absolute path to the AIP's current location on the local filesystem
    sip_uuid = UUID of the SIP, which will become the UUID of the AIP
    sip_name = SIP name.  Not used directly, but part of the AIP name
    sip_type = "SIP", "AIC", "AIP", "DIP"

    Example inputs:
    storeAIP.py
        "/var/archivematica/sharedDirectory/currentlyProcessing/ep6-0737708e-9b99-471a-b331-283e2244164f/ep6-0737708e-9b99-471a-b331-283e2244164f.7z"
        "0737708e-9b99-471a-b331-283e2244164f"
        "ep6"
        "AIP"
    """

    shared_path = "var/archivematica/sharedDirectory/"
    relative_aip_path = aip_path.replace(shared_path, "")

    # Get the package type: AIC or AIP
    if "SIP" in sip_type or "AIP" in sip_type:  # Also matches AIP-REIN
        package_type = "AIP"
    elif "AIC" in sip_type:  # Also matches AIC-REIN
        package_type = "AIC"
    elif "DIP" in sip_type:
        package_type = "DIP"

    # If AIP is a directory, calculate size recursively.
    if os.path.isdir(aip_path):
        size = 0
        for dirpath, _, filenames in os.walk(aip_path):
            for filename in filenames:
                file_path = os.path.join(dirpath, filename)
                size += os.path.getsize(file_path)
    else:
        size = os.path.getsize(aip_path)

    if "AIP" in package_type:
        metrics.aip_stored(sip_uuid, size)
    elif "DIP" in package_type:
        metrics.dip_stored(sip_uuid, size)

    return 0


def call(jobs):
    parser = argparse.ArgumentParser(description="Create AIP pointer file.")
    parser.add_argument("aip_filename", type=str, help="%AIPFilename%")
    parser.add_argument("sip_uuid", type=str, help="%SIPUUID%")
    parser.add_argument("sip_name", type=str, help="%SIPName%")
    parser.add_argument("sip_type", type=str, help="%SIPType%")

    with transaction.atomic():
        for job in jobs:
            with job.JobContext(logger=logger):
                args = parser.parse_args(job.args[1:])
                job.set_status(
                    store_aip(
                        job,
                        args.aip_filename,
                        args.sip_uuid,
                        args.sip_name,
                        args.sip_type,
                    )
                )
