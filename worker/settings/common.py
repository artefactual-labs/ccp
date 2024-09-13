# This file is part of Archivematica.
#
# Copyright 2010-2015 Artefactual Systems Inc. <http://artefactual.com>
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
import configparser
import importlib.util
import json
import logging.config
import multiprocessing
import os
from io import StringIO
from pathlib import Path

from django.core.exceptions import ImproperlyConfigured

from worker.utils import email_settings
from worker.utils.appconfig import Config


def _get_settings_from_file(path):
    spec = importlib.util.spec_from_file_location(path.stem, path)
    module = importlib.util.module_from_spec(spec)
    try:
        spec.loader.exec_module(module)
    except Exception as err:
        raise ImproperlyConfigured(f"{path} could not be imported: {err}")
    if hasattr(module, "__all__"):
        attrs = module.__all__
    else:
        attrs = [attr for attr in dir(module) if not attr.startswith("_")]
    return {attr: getattr(module, attr) for attr in attrs}


def workers(config, section):
    try:
        return config.config.getint(section, "workers")
    except (configparser.Error, ValueError):
        return multiprocessing.cpu_count()


CONFIG_MAPPING = {
    # [main]
    "workers": {
        "section": "main",
        "option": "workers",
        "process_function": workers,
    },
    "max_tasks_per_child": {
        "section": "main",
        "option": "max_tasks_per_child",
        "type": "int",
    },
    "shared_dir": {
        "section": "main",
        "option": "shared_dir",
        "type": "string",
    },
    "gearman_server": {
        "section": "main",
        "option": "gearman_server",
        "type": "string",
    },
    "capture_client_script_output": {
        "section": "main",
        "option": "capture_client_script_output",
        "type": "boolean",
    },
    "removable_files": {
        "section": "main",
        "option": "removable_files",
        "type": "string",
    },
    "secret_key": {
        "section": "main",
        "option": "django_secret_key",
        "type": "string",
    },
    "agentarchives_client_timeout": {
        "section": "main",
        "option": "agentarchives_client_timeout",
        "type": "float",
    },
    "prometheus_bind_address": {
        "section": "main",
        "option": "prometheus_bind_address",
        "type": "string",
    },
    "prometheus_bind_port": {
        "section": "main",
        "option": "prometheus_bind_port",
        "type": "string",
    },
    "prometheus_detailed_metrics": {
        "section": "main",
        "option": "prometheus_detailed_metrics",
        "type": "boolean",
    },
    "metadata_xml_validation_enabled": {
        "section": "main",
        "option": "metadata_xml_validation_enabled",
        "type": "boolean",
    },
    # [clamav]
    "clamav_server": {
        "section": "clamav",
        "option": "server",
        "type": "string",
    },
    "clamav_pass_by_stream": {
        "section": "clamav",
        "option": "pass_by_stream",
        "type": "boolean",
    },
    "clamav_client_timeout": {
        "section": "clamav",
        "option": "client_timeout",
        "type": "float",
    },
    "clamav_client_backend": {
        "section": "clamav",
        "option": "client_backend",
        "type": "string",
    },
    "clamav_client_max_file_size": {
        "section": "clamav",
        "option": "client_max_file_size",
        "type": "float",  # float for megabytes to preserve fractions on in-code operations on bytes
    },
    "clamav_client_max_scan_size": {
        "section": "clamav",
        "option": "client_max_scan_size",
        "type": "float",
    },
    # [db]
    "db_engine": {"section": "db", "option": "engine", "type": "string"},
    "db_name": {"section": "db", "option": "database", "type": "string"},
    "db_user": {"section": "db", "option": "user", "type": "string"},
    "db_password": {"section": "db", "option": "password", "type": "string"},
    "db_host": {"section": "db", "option": "host", "type": "string"},
    "db_port": {"section": "db", "option": "port", "type": "string"},
}

CONFIG_MAPPING.update(email_settings.CONFIG_MAPPING)

CONFIG_DEFAULTS = """[main]
django_secret_key =
gearman_server = localhost:4730
shared_dir = /var/archivematica/sharedDirectory/
workers =
max_tasks_per_child = 10
capture_client_script_output = true
removable_files = Thumbs.db, Icon, Icon\r, .DS_Store
metadata_xml_validation_enabled = false
agentarchives_client_timeout = 300
prometheus_bind_address =
prometheus_bind_port =
prometheus_detailed_metrics = false

[clamav]
server = /var/run/clamav/clamd.ctl
pass_by_stream = True
client_timeout = 86400
client_backend = clamscanner     ; Options: clamdscanner or clamscanner
client_max_file_size = 42        ; MB
client_max_scan_size = 42        ; MB

[db]
user = archivematica
password = demo
host = localhost
database = CCP
port = 3306
engine = django.db.backends.mysql

[email]
backend = django.core.mail.backends.console.EmailBackend
host = smtp.gmail.com
host_password =
host_user = your_email@example.com
port = 587
ssl_certfile =
ssl_keyfile =
use_ssl = False
use_tls = True
file_path =
default_from_email = webmaster@example.com
subject_prefix = [Archivematica]
timeout = 300
#server_email =
"""

config = Config(env_prefix="ARCHIVEMATICA_WORKER", attrs=CONFIG_MAPPING)
config.read_defaults(StringIO(CONFIG_DEFAULTS))
config.read_files(["/etc/archivematica/worker.conf"])


DATABASES = {
    "default": {
        "ENGINE": config.get("db_engine"),
        "NAME": config.get("db_name"),
        "USER": config.get("db_user"),
        "PASSWORD": config.get("db_password"),
        "HOST": config.get("db_host"),
        "PORT": config.get("db_port"),
        "CONN_MAX_AGE": 3600,  # 1 hour
    }
}

# Make this unique, and don't share it with anybody.
SECRET_KEY = config.get("secret_key")

USE_TZ = True
TIME_ZONE = "UTC"

INSTALLED_APPS = (
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "worker.main",
    "worker.legacy.administration",
    "worker.fpr",
    # Only needed because archivematicaClient calls django.setup()
    # which imports the ApiAccess model through the helpers module of
    # the dashboard
    "tastypie",
)

# Configure logging manually
LOGGING_CONFIG = None

# Location of the logging configuration file that we're going to pass to
# `logging.config.fileConfig` unless it doesn't exist.
LOGGING_CONFIG_FILE = "/etc/archivematica/clientConfig.logging.json"

LOGGING = {
    "version": 1,
    "disable_existing_loggers": False,
    "formatters": {
        "detailed": {
            "format": "%(levelname)-8s  %(asctime)s  %(name)s:%(module)s:%(funcName)s:%(lineno)d:  %(message)s",
            "datefmt": "%Y-%m-%d %H:%M:%S",
        }
    },
    "handlers": {
        "console": {
            "level": "DEBUG",
            "class": "logging.StreamHandler",
            "formatter": "detailed",
        }
    },
    "loggers": {"archivematica": {"level": "DEBUG"}},
    "root": {"handlers": ["console"], "level": "WARNING"},
}

if os.path.isfile(LOGGING_CONFIG_FILE):
    with open(LOGGING_CONFIG_FILE) as f:
        logging.config.dictConfig(json.load(f))
else:
    logging.config.dictConfig(LOGGING)


SHARED_DIRECTORY = os.path.join(config.get("shared_dir"), "")
PROCESSING_DIRECTORY = os.path.join(SHARED_DIRECTORY, "currentlyProcessing", "")
REJECTED_DIRECTORY = os.path.join(SHARED_DIRECTORY, "rejected", "")
WATCH_DIRECTORY = os.path.join(SHARED_DIRECTORY, "watchedDirectories", "")
TEMP_DIRECTORY = os.path.join(SHARED_DIRECTORY, "tmp")

WORKERS = config.get("workers")
MAX_TASKS_PER_CHILD = config.get("max_tasks_per_child")
GEARMAN_SERVER = config.get("gearman_server")
REMOVABLE_FILES = config.get("removable_files")

# [clamav]
CLAMAV_SERVER = config.get("clamav_server")
CLAMAV_PASS_BY_STREAM = config.get("clamav_pass_by_stream")
CLAMAV_CLIENT_TIMEOUT = config.get("clamav_client_timeout")
CLAMAV_CLIENT_BACKEND = config.get("clamav_client_backend")
CLAMAV_CLIENT_MAX_FILE_SIZE = config.get("clamav_client_max_file_size")
CLAMAV_CLIENT_MAX_SCAN_SIZE = config.get("clamav_client_max_scan_size")

AGENTARCHIVES_CLIENT_TIMEOUT = config.get("agentarchives_client_timeout")
CAPTURE_CLIENT_SCRIPT_OUTPUT = config.get("capture_client_script_output")
DEFAULT_CHECKSUM_ALGORITHM = "sha256"

PROMETHEUS_DETAILED_METRICS = config.get("prometheus_detailed_metrics")
PROMETHEUS_BIND_ADDRESS = config.get("prometheus_bind_address")
try:
    PROMETHEUS_BIND_PORT = int(config.get("prometheus_bind_port"))
except ValueError:
    PROMETHEUS_ENABLED = False
else:
    PROMETHEUS_ENABLED = True

TEMPLATES = [{"BACKEND": "django.template.backends.django.DjangoTemplates"}]

# Apply email settings
globals().update(email_settings.get_settings(config))

METADATA_XML_VALIDATION_ENABLED = config.get("metadata_xml_validation_enabled")
if METADATA_XML_VALIDATION_ENABLED:
    METADATA_XML_VALIDATION_SETTINGS_FILE = os.environ.get(
        "ARCHIVEMATICA_WORKER_METADATA_XML_VALIDATION_SETTINGS_FILE", ""
    )
    if METADATA_XML_VALIDATION_SETTINGS_FILE:
        xml_validation_settings = _get_settings_from_file(
            Path(METADATA_XML_VALIDATION_SETTINGS_FILE)
        )
        XML_VALIDATION = xml_validation_settings.get("XML_VALIDATION")
        XML_VALIDATION_FAIL_ON_ERROR = xml_validation_settings.get(
            "XML_VALIDATION_FAIL_ON_ERROR"
        )
        if not isinstance(XML_VALIDATION, dict) or not isinstance(
            XML_VALIDATION_FAIL_ON_ERROR, bool
        ):
            raise ImproperlyConfigured(
                f"The metadata XML validation settings file {METADATA_XML_VALIDATION_SETTINGS_FILE} does not contain "
                "the right settings: an XML_VALIDATION dictionary and an XML_VALIDATION_FAIL_ON_ERROR boolean"
            )
