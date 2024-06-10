import logging
from collections import namedtuple

from client.base import Client
from client.gen.archivematica.ccp.admin.v1beta1 import admin_pb2
from client.gen.archivematica.ccp.admin.v1beta1 import deprecated_pb2
from client.gen.archivematica.ccp.admin.v1beta1 import service_pb2
from client.gen.archivematica.ccp.admin.v1beta1 import service_pb2_grpc
from django.conf import settings
from google.protobuf.wrappers_pb2 import BoolValue
from google.protobuf.wrappers_pb2 import StringValue
from grpc import ClientCallDetails
from grpc import StreamStreamClientInterceptor
from grpc import StreamUnaryClientInterceptor
from grpc import UnaryStreamClientInterceptor
from grpc import UnaryUnaryClientInterceptor
from grpc import insecure_channel
from grpc import intercept_channel
from lxml import etree

LOGGER = logging.getLogger("archivematica.dashboard.mcp.client")

TRANSFER_TYPES = {
    "standard": admin_pb2.TRANSFER_TYPE_STANDARD,
    "zipfile": admin_pb2.TRANSFER_TYPE_ZIP_FILE,
    "unzipped bag": admin_pb2.TRANSFER_TYPE_UNZIPPED_BAG,
    "zipped bag": admin_pb2.TRANSFER_TYPE_ZIPPED_BAG,
    "dspace": admin_pb2.TRANSFER_TYPE_DSPACE,
    "maildir": admin_pb2.TRANSFER_TYPE_MAILDIR,
    "TRIM": admin_pb2.TRANSFER_TYPE_TRIM,
    "dataverse": admin_pb2.TRANSFER_TYPE_DATAVERSE,
}

PACKAGE_TYPES = {
    "Transfer": admin_pb2.PACKAGE_TYPE_TRANSFER,
    "SIP": admin_pb2.PACKAGE_TYPE_SIP,
}


class GenericClientInterceptor(
    UnaryUnaryClientInterceptor,
    UnaryStreamClientInterceptor,
    StreamUnaryClientInterceptor,
    StreamStreamClientInterceptor,
):
    def __init__(self, interceptor_function):
        self._fn = interceptor_function

    def intercept_unary_unary(self, continuation, client_call_details, request):
        new_details, new_request_iterator, postprocess = self._fn(
            client_call_details, iter((request,)), False, False
        )
        response = continuation(new_details, next(new_request_iterator))
        return postprocess(response) if postprocess else response

    def intercept_unary_stream(self, continuation, client_call_details, request):
        new_details, new_request_iterator, postprocess = self._fn(
            client_call_details, iter((request,)), False, True
        )
        response_it = continuation(new_details, next(new_request_iterator))
        return postprocess(response_it) if postprocess else response_it

    def intercept_stream_unary(
        self, continuation, client_call_details, request_iterator
    ):
        new_details, new_request_iterator, postprocess = self._fn(
            client_call_details, request_iterator, True, False
        )
        response = continuation(new_details, new_request_iterator)
        return postprocess(response) if postprocess else response

    def intercept_stream_stream(
        self, continuation, client_call_details, request_iterator
    ):
        new_details, new_request_iterator, postprocess = self._fn(
            client_call_details, request_iterator, True, True
        )
        response_it = continuation(new_details, new_request_iterator)
        return postprocess(response_it) if postprocess else response_it


def create_generic_interceptor(intercept_call):
    return GenericClientInterceptor(intercept_call)


class _ClientCallDetails(
    namedtuple("_ClientCallDetails", ("method", "timeout", "metadata", "credentials")),
    ClientCallDetails,
):
    pass


def header_adder_interceptor(headers):
    def intercept_call(
        client_call_details,
        request_iterator,
        request_streaming,
        response_streaming,
    ):
        metadata = []
        if client_call_details.metadata is not None:
            metadata = list(client_call_details.metadata)
        for pair in headers:
            metadata.append(pair)
        client_call_details = _ClientCallDetails(
            client_call_details.method,
            client_call_details.timeout,
            metadata,
            client_call_details.credentials,
        )
        return client_call_details, request_iterator, None

    return create_generic_interceptor(intercept_call)


class CCPClient(Client):
    """CCPClient is a gRPC client of the Admin API.

    The Admin API is a Connect server with multi-protocol support: gRPC,
    gRPC-Weband Connect's own protocol. Since a Connect client is not officially
    supported yet, we choose to make use of the gRPC protocol.
    """

    def __init__(self, user_id, lang):
        self.user_id = str(user_id)
        self.lang = lang
        self.channel = insecure_channel(
            target=settings.GEARMAN_SERVER,
            options=[
                ("grpc.lb_policy_name", "pick_first"),
                ("grpc.enable_retries", 0),
                ("grpc.keepalive_timeout_ms", 10000),
            ],
        )
        self.channel = intercept_channel(
            self.channel,
            header_adder_interceptor(
                [
                    ("user_id", self.user_id),
                    ("lang", self.lang),
                ],
            ),
        )
        self.stub = service_pb2_grpc.AdminServiceStub(self.channel)

    def _tx(self, translations):
        tx = translations.tx

        return tx.get(self.lang, tx.get("en", ""))

    def approve_job(self, job_id, choice):
        """Uses AdminService.ApproveJob (deprecated)."""
        req = deprecated_pb2.ApproveJobRequest(job_id=job_id, choice=choice)
        self.stub.ApproveJob(req)
        return "approving: {job_id} {choice}"

    def list_jobs_awaiting_approval(self):
        """Uses AdminService.ListDecisions."""
        req = service_pb2.ListDecisionsRequest()
        resp = self.stub.ListDecisions(req)

        def choices_available_for_unit(tree, decision):
            obj = etree.SubElement(tree, "choicesAvailableForUnit")
            unit = etree.SubElement(obj, "unit")
            etree.SubElement(obj, "UUID").text = decision.id
            etree.SubElement(unit, "type").text = decision.package_type
            unitXML = etree.SubElement(unit, "unitXML")
            etree.SubElement(unitXML, "UUID").text = str(decision.package_id)
            etree.SubElement(unitXML, "currentPath").text = decision.package_path
            choices = etree.SubElement(obj, "choices")
            for item in decision.choice:
                choice = etree.SubElement(choices, "choice")
                etree.SubElement(choice, "chainAvailable").text = str(item.id)
                etree.SubElement(choice, "description").text = item.label

        ret = etree.Element("choicesAvailableForUnits")
        for decision in resp.decision:
            choices_available_for_unit(ret, decision)

        return etree.tostring(ret, pretty_print=True, encoding="utf8")

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
        """Uses AdminService.CreatePackage."""
        transfer_type = admin_pb2.PACKAGE_STATUS_PROCESSING
        req = service_pb2.CreatePackageRequest(
            name=name,
            type=TRANSFER_TYPES.get(transfer_type),
            accession=accession,
            access_system_id=access_system_id,
            auto_approve=BoolValue(value=auto_approve),
            processing_config=processing_config,
        )
        if isinstance(path, (list, set)):
            req.path.extend(path)
        elif isinstance(path, str):
            req.path.append(path)
        if len(metadata_set_id) > 0:
            req.metadata_set_id = StringValue(value=metadata_set_id)
        return self.stub.CreatePackage(req).id

    def approve_transfer_by_path(self, db_transfer_path, transfer_type):
        """Uses AdminService.ApproveTransferByPath (deprecated)."""
        req = deprecated_pb2.ApproveTransferByPathRequest(
            directory=db_transfer_path,
            type=TRANSFER_TYPES.get(transfer_type),
        )
        resp = self.stub.ApproveTransferByPath(req)

        return resp.id

    def approve_partial_reingest(self, sip_id):
        """Uses AdminService.ApprovePartialReingest (deprecated)."""
        req = deprecated_pb2.ApprovePartialReingestRequest(id=sip_id)
        self.stub.ApprovePartialReingest(req)

    def get_processing_config_fields(self):
        """Uses AdminService.ListProcessingConfigurationFields."""
        req = service_pb2.ListProcessingConfigurationFieldsRequest()
        resp = self.stub.ListProcessingConfigurationFields(req)

        def format_applies_to(applies_to):
            return (
                applies_to.link_id,
                applies_to.value,
                self._tx(applies_to.label),
            )

        def format_choice(choice):
            return {
                "value": choice.value,
                "label": self._tx(choice.label),
                "applies_to": [
                    format_applies_to(applies_to) for applies_to in choice.applies_to
                ],
            }

        def format_field(field):
            return {
                "id": field.id,
                "name": field.name,
                "label": self._tx(field.label),
                "choices": [format_choice(choice) for choice in field.choice],
            }

        return [format_field(field) for field in resp.field]

    def get_packages_status(self, package_type):
        """Uses AdminService.ListActivePackages."""
        package_type = PACKAGE_TYPES.get(package_type)
        req = service_pb2.ListPackagesRequest(type=package_type)
        resp = self.stub.ListPackages(req)

        def format_pkg(pkg):
            ret = {
                "id": pkg.id,
                "uuid": pkg.id,
                "timestamp": convert_wkt_timestamp_to_float(pkg.created_at),
                "active": pkg.status == admin_pb2.PACKAGE_STATUS_PROCESSING,
                "directory": pkg.name,
                "jobs": [format_job(job) for job in pkg.job],
            }
            if len(pkg.access_system_id) > 0:
                ret["access_system_id"] = pkg.access_system_id
            return ret

        def format_job(job):
            ret = {
                "uuid": job.id,
                "link_id": job.link_id,
                "currentstep": job.status,
                "timestamp": convert_wkt_timestamp_to_string(job.created_at),
                "microservicegroup": job.group,
                "type": job.link_description,
            }
            if job.HasField("decision"):
                ret["choices"] = format_choices(job.decision)
            return ret

        def format_choices(decision):
            return {choice.id: choice.label for choice in decision.choice}

        ret = [format_pkg(pkg) for pkg in resp.package]

        return ret

    def get_package_status(self, id):
        """Uses AdminService.ReadPackage."""
        req = service_pb2.ReadPackageRequest(id=id)
        resp = self.stub.ReadPackage(req)
        # TODO: format response.
        return resp

    def close(self):
        self.channel.close()


def convert_wkt_timestamp_to_float(ts):
    ns_per_sec = 1000000000
    return ts.seconds + ts.nanos / ns_per_sec


def convert_wkt_timestamp_to_string(ts):
    combined = convert_wkt_timestamp_to_float(ts)
    return f"{combined:.10f}"
