"""

Reminders:

- [*] Validate configuration.
- [ ] Use mypy in VSCode
- [ ] Douglas thinks that we should review what we send to pyprint and logger.

Testing:

  $ pytest --disable-warnings -pno:randomly -vv --cov --cov-report html:/tmp/coverage.html -x -- tests/test_antivirus.py
  $ open file://///tmp/coverage.html/index.html
  $ mypy tests/test_antivirus.py worker/clientScripts/archivematica_clamscan.py

Misc:

- [ ] mypy pre-commit hook isn't working, move to dagger?
      just -> dagger (for everything)
- [ ] dagger to test with mysql (sqlite isn't really supported)

"""

from multiprocessing import cpu_count
from unittest import mock

import pytest
import pytest_django

from worker.client.job import Job
from worker.clientScripts.archivematica_clamscan import call
from worker.clientScripts.archivematica_clamscan import concurrent_instances
from worker.main.models import Event
from worker.main.models import File

TASK_DATE = "2024-09-13T09:17:35.702089+00:00"  # isoformat


def test_concurrent_instances() -> None:
    assert concurrent_instances() == cpu_count()


def test_invalid_config_backend(
    settings: pytest_django.fixtures.SettingsWrapper,
) -> None:
    settings.CLAMAV_CLIENT_BACKEND = "invalid-backend"
    with pytest.raises(ValueError, match="Unsupported backend type: invalid-backend"):
        call([])


@pytest.mark.django_db
def test_valid_backends(
    settings: pytest_django.fixtures.SettingsWrapper,
) -> None:
    for backend in (
        "clamscanner",  # » ``clamav_client.scanner.ClamscanScanner``.
        "clamscan",  # » ``clamav_client.scanner.ClamscanScanner``.
        "clamdscanner",  # » ``clamav_client.scanner.ClamdScanner``.
        "clamd",  # » ``clamav_client.scanner.ClamdScanner``.
    ):
        settings.CLAMAV_CLIENT_BACKEND = backend
        call([])


@pytest.mark.django_db
def test_invalid_file_id() -> None:
    job = mock.Mock(
        args=[
            "archivematica_clamscan.py",
            "None",
            "path",
            TASK_DATE,
        ],
        JobContext=mock.MagicMock(),
        spec=Job,
    )

    call([job])

    job.set_status.assert_called_once_with(1)

    assert Event.objects.filter(event_type="virus check").count() == 0


@pytest.mark.django_db
def test_file_already_scanned(transfer_file: File) -> None:
    Event.objects.create(file_uuid=transfer_file, event_type="virus check", event_outcome="Pass")

    job = mock.Mock(
        args=[
            "archivematica_clamscan.py",
            str(transfer_file.pk),
            transfer_file.currentlocation.decode(),
            TASK_DATE,
        ],
        JobContext=mock.MagicMock(),
        spec=Job,
    )

    call([job])

    job.set_status.assert_called_once_with(0)
    assert (
        Event.objects.filter(file_uuid=transfer_file, event_type="virus check").count()
        == 1
    )


@pytest.mark.django_db
def test_unsized_file(transfer_file: File) -> None:
    job = mock.Mock(
        args=[
            "archivematica_clamscan.py",
            str(transfer_file.pk),
            transfer_file.currentlocation.decode(),
            TASK_DATE,
        ],
        JobContext=mock.MagicMock(),
        spec=Job,
    )

    call([job])

    job.set_status.assert_called_once_with(1)
    assert (
        Event.objects.filter(
            file_uuid=transfer_file, event_type="virus check", event_outcome="Fail"
        ).count()
        == 1
    )


@pytest.mark.django_db
def test_unexistent_file() -> None:
    job = mock.Mock(
        args=[
            "archivematica_clamscan.py",
            "6cc38bf0-d8e2-414f-a8f8-c946ae2b5255",
            "/path",
            TASK_DATE,
        ],
        JobContext=mock.MagicMock(),
        spec=Job,
    )

    call([job])

    job.set_status.assert_called_once_with(1)
    assert Event.objects.filter(event_type="virus check").count() == 0


@pytest.mark.django_db
def test_file_size_exceeding_max_settings() -> None:
    pass  # TODO


@pytest.mark.django_db
def test_scan_exception() -> None:
    pass  # TODO


@pytest.mark.django_db
def test_scan_passed() -> None:
    pass  # TODO


@pytest.mark.django_db
def test_scan_found() -> None:
    pass  # TODO


@pytest.mark.django_db
def test_scan_many() -> None:
    pass  # TODO
