import abc

from main.models import Job


class NoJobFoundError(Exception):
    def __init__(self, message=None):
        if message is None:
            message = "No job was found"
        super().__init__(message)


class Client(abc.ABC):
    """Handles communication with the server."""

    @abc.abstractmethod
    def approve_job(self, job_id, choice):
        pass

    def execute_package(self, unit_id, choice, mscl_id=None):
        """Execute the jobs awaiting for approval associated to a given package.

        Use ``mscl_id`` to pass the ID of the chain link to restrict the
        execution to a single microservice.
        """
        kwargs = {"currentstep": Job.STATUS_AWAITING_DECISION, "sipuuid": unit_id}
        if mscl_id is not None:
            kwargs["microservicechainlink"] = mscl_id
        jobs = Job.objects.filter(**kwargs)
        if len(jobs) < 1:
            raise NoJobFoundError()
        for item in jobs:
            self.approve_job(item.pk, choice)

    @abc.abstractmethod
    def list_jobs_awaiting_approval(self):
        pass

    @abc.abstractmethod
    def create_package(
        self,
        name,
        type,
        accession,
        access_system_id,
        path,
        metadata_set_id,
        auto_approve=True,
        wait_until_complete=False,
        processing_config=None,
    ):
        pass

    @abc.abstractmethod
    def approve_transfer_by_path(self, db_transfer_path, transfer_type):
        pass

    @abc.abstractmethod
    def approve_partial_reingest(self, sip_id):
        pass

    @abc.abstractmethod
    def get_processing_config_fields(self):
        pass

    @abc.abstractmethod
    def get_packages_status(self, package_type):
        pass

    def get_transfers_status(self):
        return self.get_packages_status("Transfer")

    def get_sips_status(self):
        return self.get_packages_status("SIP")

    @abc.abstractmethod
    def get_package_status(self, id):
        pass

    @abc.abstractmethod
    def close(self):
        pass
