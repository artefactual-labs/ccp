#!/usr/bin/env python
import argparse

import django

django.setup()
from client import metrics
from django.db import transaction
from main.models import File
from main.models import Transfer

REJECTED = "reject"
FAILED = "fail"


def main(job, fail_type, transfer_uuid, transfer_path):
    # Delete files for reingest transfer
    # A new reingest doesn't know to delete this because the UUID is different
    # from the AIP, and it causes problems when re-parsing these files.
    transfer = Transfer.objects.get(uuid=transfer_uuid)
    if transfer.type == "Archivematica AIP":
        File.objects.filter(transfer_id=transfer_uuid).delete()

    metrics.transfer_failed(transfer.type, fail_type)

    return 0


def call(jobs):
    parser = argparse.ArgumentParser(
        description="Cleanup from failed/rejected Transfers."
    )
    parser.add_argument("fail_type", help=f'"{REJECTED}" or "{FAILED}"')
    parser.add_argument("transfer_uuid", help="%SIPUUID%")
    parser.add_argument("transfer_path", help="%SIPDirectory%")

    with transaction.atomic():
        for job in jobs:
            with job.JobContext():
                args = parser.parse_args(job.args[1:])
                job.set_status(
                    main(job, args.fail_type, args.transfer_uuid, args.transfer_path)
                )
