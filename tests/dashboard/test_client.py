from unittest.mock import MagicMock

import client
import pytest
from client import CCPClient
from client import MCPClient
from client import get_client
from client.gen.archivematica.ccp.admin.v1beta1.admin_pb2 import CreatePackageResponse
from gearman import JOB_COMPLETE

USER_ID = "12345"


class GearmanSubmitJobResult:
    def __init__(self, state, result):
        self.state = state
        self.result = self.dict_to_object(result)

    @classmethod
    def dict_to_object(cls, d):
        if not isinstance(d, dict):
            return d
        return type(
            "DynamicObject", (object,), {k: cls.dict_to_object(v) for k, v in d.items()}
        )


@pytest.fixture(autouse=True)
def reset_instance():
    client._client_instance = None
    yield


@pytest.fixture
def patch_gearman_client(mocker):
    return mocker.patch("client.mcp.GearmanClient")


@pytest.fixture
def patch_admin_stub(mocker):
    return mocker.patch(
        "client.gen.archivematica.ccp.admin.v1beta1.admin_pb2_grpc.AdminServiceStub"
    )


@pytest.mark.parametrize(
    "client_class,expected",
    [
        (None, CCPClient),
        (MCPClient, MCPClient),
        (CCPClient, CCPClient),
    ],
)
def test_client_getter(client_class, expected):
    client = get_client(USER_ID, client_class)
    assert isinstance(client, expected)


def test_mcp_client_approve_job(patch_gearman_client):
    gearman_client = MagicMock()
    gearman_client.submit_job.return_value = GearmanSubmitJobResult(
        state=JOB_COMPLETE,
        result="approving: ...",
    )
    patch_gearman_client.return_value = gearman_client

    client = get_client(USER_ID, client_class=MCPClient)
    resp = client.approve_job("job_id", "choice")

    assert resp == "approving: ..."


def test_mcp_client_create_package(patch_gearman_client):
    gearman_client = MagicMock()
    gearman_client.submit_job.return_value = GearmanSubmitJobResult(
        state=JOB_COMPLETE,
        result="c3a3ec8b-adbb-4787-8047-3d2841eae911",
    )
    patch_gearman_client.return_value = gearman_client

    client = get_client(USER_ID, client_class=MCPClient)
    resp = client.create_package(
        "name",
        "Transfer",
        "accession",
        "access_sytem_id",
        "path",
        "metadata_set_id",
    )

    assert resp == "c3a3ec8b-adbb-4787-8047-3d2841eae911"


def test_ccp_client_create_package(patch_admin_stub):
    stub = patch_admin_stub.return_value
    stub.CreatePackage = MagicMock(
        return_value=CreatePackageResponse(id="c3a3ec8b-adbb-4787-8047-3d2841eae911")
    )

    client = get_client(USER_ID, client_class=CCPClient)
    resp = client.create_package(
        "name",
        "Transfer",
        "accession",
        "access_sytem_id",
        "path",
        "metadata_set_id",
    )

    assert resp == "c3a3ec8b-adbb-4787-8047-3d2841eae911"
