#!/usr/bin/env python
import argparse

import django
from django.db import transaction

django.setup()

from worker.client import metrics
from worker.main import models

REJECTED = "reject"
FAILED = "fail"


def main(job, fail_type, sip_uuid):
    # Update SIP Arrange table for failed SIP
    file_uuids = models.File.objects.filter(sip=sip_uuid).values_list("uuid", flat=True)
    job.pyprint("Allow files in this SIP to be arranged. UUIDs:", file_uuids)
    models.SIPArrange.objects.filter(sip_id=sip_uuid).delete()

    return 0


def call(jobs):
    parser = argparse.ArgumentParser(description="Cleanup from failed/rejected SIPs.")
    parser.add_argument("fail_type", help=f'"{REJECTED}" or "{FAILED}"')
    parser.add_argument("sip_uuid", help="%SIPUUID%")

    with transaction.atomic():
        for job in jobs:
            with job.JobContext():
                args = parser.parse_args(job.args[1:])
                job.set_status(main(job, args.fail_type, args.sip_uuid))

    metrics.sip_failed(args.fail_type)
