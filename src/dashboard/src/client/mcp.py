import logging

import gearman
from client.base import Client
from django.conf import settings
from gearman_encoder import JSONDataEncoder

LOGGER = logging.getLogger("archivematica.dashboard.mcp.client")


class RPCGearmanClientError(Exception):
    """Base exception."""


class RPCError(RPCGearmanClientError):
    """Unexpected error."""


class RPCServerError(RPCGearmanClientError):
    """Server application errors.

    When the worker processes the job successfully but the response includes
    an error.
    """

    GENERIC_ERROR_MSG = "The server failed to process the request"

    def __init__(self, payload=None):
        super().__init__(self._process_error(payload))

    def _process_error(self, payload):
        """Extracts the error message from the payload."""
        if payload is None or not isinstance(payload, dict):
            return self.GENERIC_ERROR_MSG
        message = payload.get("message", "Unknown error message")
        handler = payload.get("function")
        if handler:
            message += f" [handler={handler}]"
        return message


class TimeoutError(RPCGearmanClientError):
    """Deadline exceeded.

    >> response = client.submit_job(
           "doSomething", data,
           background=False, wait_until_complete=True,
           poll_timeout=INFLIGHT_POLL_TIMEOUT)
       if response.state == gearman.JOB_CREATED:
           raise TimeoutError()

    At this point we give up and raise this exception.
    """

    def __init__(self, timeout=None):
        message = "Deadline exceeded"
        if timeout is not None:
            message = f"{message}: {timeout}"
        super().__init__(message)


class GearmanClient(gearman.GearmanClient):
    data_encoder = JSONDataEncoder


INFLIGHT_POLL_TIMEOUT = 30.0


class MCPClient(Client):
    """Handles communication with MCPServer."""

    def __init__(self, user_id, lang):
        self.server = settings.GEARMAN_SERVER
        self.user_id = user_id
        self.lang = lang

    def _rpc_sync_call(self, ability, data=None, timeout=INFLIGHT_POLL_TIMEOUT):
        """Invoke remote method synchronously and with a deadline.

        When successful, it returns the payload of the response. Otherwise, it
        raises an exception. ``TimeoutError`` when the deadline was exceeded,
        ``RPCError`` when the worker failed abruptly, ``RPCServerError`` when
        the worker returned an error.
        """
        if data is None:
            data = b""
        elif "user_id" not in data:
            data["user_id"] = self.user_id
        client = GearmanClient([self.server])
        response = client.submit_job(
            ability.encode(),
            data,
            background=False,
            wait_until_complete=True,
            poll_timeout=timeout,
        )
        client.shutdown()
        if response.state == gearman.JOB_CREATED:
            raise TimeoutError(timeout)
        elif response.state != gearman.JOB_COMPLETE:
            raise RPCError(f"{ability} failed (check the logs)")
        payload = response.result
        if isinstance(payload, dict) and payload.get("error", False):
            raise RPCServerError(payload)
        return payload

    def approve_job(self, job_id, choice):
        return self._rpc_sync_call(
            "approveJob",
            {
                "jobUUID": job_id,
                "chain": choice,
            },
        )

    def list_jobs_awaiting_approval(self):
        return self._rpc_sync_call("getJobsAwaitingApproval", {})

    def create_package(
        self,
        name,
        transfer_type,
        accession,
        access_system_id,
        path,
        metadata_set_id,
        auto_approve=True,
        wait_until_complete=False,
        processing_config=None,
    ):
        data = {
            "name": name,
            "type": transfer_type,
            "accession": accession,
            "access_system_id": access_system_id,
            "path": path,
            "metadata_set_id": metadata_set_id,
            "auto_approve": auto_approve,
            "wait_until_complete": wait_until_complete,
        }
        if processing_config is not None:
            data["processing_config"] = processing_config
        return self._rpc_sync_call("packageCreate", data)

    def approve_transfer_by_path(self, db_transfer_path, transfer_type):
        return self._rpc_sync_call(
            "approveTransferByPath",
            {
                "db_transfer_path": db_transfer_path,
                "transfer_type": transfer_type,
            },
        )

    def approve_partial_reingest(self, sip_id):
        return self._rpc_sync_call(
            "approvePartialReingest",
            {
                "sip_id": sip_id,
            },
        )

    def get_processing_config_fields(self):
        return self._rpc_sync_call(
            "getProcessingConfigFields",
            {
                "lang": self.lang,
            },
        )

    def get_packages_status(self, package_type):
        return self._rpc_sync_call(
            "getUnitsStatuses",
            {
                "type": package_type,
                "lang": self.lang,
            },
        )

    def get_package_status(self, id):
        return self._rpc_sync_call(
            "getUnitStatus",
            {
                "id": id,
                "lang": self.lang,
            },
        )

    def close(self):
        pass
