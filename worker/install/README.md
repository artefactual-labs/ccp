# Worker Configuration

## Table of contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Introduction](#introduction)
- [Environment variables](#environment-variables)
- [Configuration file](#configuration-file)
- [Parameter list](#parameter-list)
  - [Metadata XML validation variables](#metadata-xml-validation-variables)
- [Logging configuration](#logging-configuration)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Introduction

Archivematica components can be configured using multiple methods. All
components follow the same pattern:

1. **Environment variables** - setting a configuration parameter with an
   environment variable will override all other methods.
1. **Configuration file** - if the parameter is not set by an environment
   variable, the component will look for a setting in its default configuration file.
1. **Application defaults** - if the parameter is not set in an environment
   variable or the config file, the application default is used.

Logging behaviour is configured differently, and provides two methods:

1. **`logging.json` file** - if a JSON file is present in the default location,
   the contents of the JSON file will control the components logging behaviour.
1. **Application default** - if no JSON file is present, the default logging
   behaviour is to write to standard streams (standard out and standard error).

## Environment variables

The value of an environment variable is a string of characters. The
configuration system coerces the value to the types supported:

- `string` (e.g. `"foobar"`)
- `int` (e.g. `"60"`)
- `float` (e.g. `"1.20"`)
- `boolean` where truth values can be represented as follows (checked in a
  case-insensitive manner):
  - True (enabled):  `"1"`, `"yes"`, `"true"` or `"on"`
  - False (disabled): `"0"`, `"no"`, `"false"` or `"off"`

Certain environment strings are mandatory, i.e. they don't have defaults and
the application will refuse to start if the user does not provide one.

Please be aware that Archivematica supports different types of distributions
(Ubuntu/CentOS packages, Ansible or Docker images) and they may override some
of these settings or provide values to mandatory fields.

## Configuration file

There is an example configuration file for this worker included in the source
code: (see [`the example`](./worker.conf)).

The worker will look for a configuration file in the following location:

    /etc/archivematica/worker.conf

## Parameter list

This is the full list of variables supported by the worker:

- **`ARCHIVEMATICA_WORKER_DJANGO_SECRET_KEY`**:
  - **Description:** a secret key used for cryptographic signing. See [SECRET_KEY]
    for more details.
  - **Config file example:** `main.django_secret_key`
  - **Type:** `string`
  - :red_circle: **Mandatory!**

- **`ARCHIVEMATICA_WORKER_WORKERS`**:
  - **Description:** number of workers. If undefined, it defaults to the
    number of CPUs available on the machine. Only client modules that define
    `concurrent_instances` will perform concurrent execution of tasks.
  - **Config file example:** `main.workers`
  - **Type:** `int`

- **`ARCHIVEMATICA_WORKER_MAX_TASKS_PER_CHILD`**:
  - **Description:** maximum number of tasks a worker can execute before it's
    replaced by a new process in order to free resources.
  - **Config file example:** `main.max_tasks_per_child`
  - **Type:** `int`
  - **Default:** `10`

- **`ARCHIVEMATICA_WORKER_SHARED_DIR`**:
  - **Description:** location of the Archivematica Shared Directory.
  - **Config file example:** `main.shared_dir`
  - **Type:** `string`
  - **Default:** `/var/archivematica/sharedDirectory/`

- **`ARCHIVEMATICA_WORKER_GEARMAN_SERVER`**:
  - **Description:** address of the Gearman server.
  - **Config file example:** `main.gearman_server`
  - **Type:** `string`
  - **Default:** `localhost:4730`

- **`ARCHIVEMATICA_WORKER_REMOVABLE_FILES`**:
  - **Description:** comma-separated listed of file names that will be deleted.
  - **Config file example:** `main.removable_files`
  - **Type:** `string`
  - **Default:** `Thumbs.db, Icon, Icon\r, .DS_Store`

- **`ARCHIVEMATICA_WORKER_AGENTARCHIVES_CLIENT_TIMEOUT`**:
  - **Description:** configures the agentarchives client to stop waiting for a
    response after a given number of seconds.
  - **Config file example:** `main.agentarchives_client_timeout`
  - **Type:** `float`
  - **Default:** `300`

- **`ARCHIVEMATICA_WORKER_CAPTURE_CLIENT_SCRIPT_OUTPUT`**:
  - **Description:** controls whether or not to capture stdout from client
    script subprocesses.  If set to `true`, then stdout is captured; if set to
    `false`, then stdout is not captured. If set to `true`, then stderr is
    captured; if set to `false`, then stderr is captured only if the subprocess
    has failed, i.e., returned a non-zero exit code.
  - **Config file example:** `main.capture_client_script_output`
  - **Type:** `boolean`
  - **Default:** `true`

- **`ARCHIVEMATICA_WORKER_METADATA_XML_VALIDATION_ENABLED`**:
  - **Description:** (**Experimental**) Determines if the XML files in the
    `metadata` directory of a SIP should be validated against a set of XML
    schemas, recorded in the METS file and indexed as custom metadata. See the
    [feature variables](#metadata-xml-validation-variables) section below.
  - **Config file example:** `main.metadata_xml_validation_enabled`
  - **Type:** `boolean`
  - **Default:** `false`

Prometheus metrics server:

- **`ARCHIVEMATICA_WORKER_PROMETHEUS_BIND_ADDRESS`**:
  - **Description:** when set to a non-empty string, its value is parsed as the
    IP address on which to serve Prometheus metrics. If this value is not
    provided, but ``ARCHIVEMATICA_WORKER_PROMETHEUS_BIND_PORT`` is, then
    `127.0.0.1` will be used.
  - **Config file example:** `main.prometheus_bind_addresss`
  - **Type:** `string`
  - **Default:** `""`

- **`ARCHIVEMATICA_WORKER_PROMETHEUS_BIND_PORT`**:
  - **Description:** The port on which to serve Prometheus metrics.
    If this value is not provided, metrics will not be served.
  - **Config file example:** `main.prometheus_bind_port`
  - **Type:** `int`
  - **Default:** `""`

- **`ARCHIVEMATICA_WORKER_PROMETHEUS_DETAILED_METRICS`**:
  - **Description:** Send detailed metrics to Prometheus. With large transfers
    this might affect performance of the local storage in Prometheus and slow
    down its threads in Archivematica.
  - **Config file example:** `main.prometheus_detailed_metrics`
  - **Type:** `boolean`
  - **Default:** `false`

Antivirus (ClamAV):

- **`ARCHIVEMATICA_WORKER_CLAMAV_SERVER`**:
  - **Description:** configures the `clamdscanner` backend so it knows how to
    reach the clamd server via UNIX socket (if the value starts with /) or TCP
    socket (form `host:port`, e.g.: `myclamad:3310`).
  - **Config file example:** `clamav.server`
  - **Type:** `string`
  - **Default:** `/var/run/clamav/clamd.ctl`

- **`ARCHIVEMATICA_WORKER_CLAMAV_PASS_BY_STREAM`**:
  - **Description:** configures the `clamdscanner` backend to stream the file's
    contents to clamd. This is useful when clamd does not have access to the
    filesystem where the file is stored. When disabled, the files are read from
    the filesystem by reference.
  - **Config file example:** `clamav.pass_by_stream`
  - **Type:** `boolean`
  - **Default:** `true`

- **`ARCHIVEMATICA_WORKER_CLAMAV_CLIENT_TIMEOUT`**:
  - **Description:** configures the `clamdscanner` backend to stop waiting for a
    response after a given number of seconds.
  - **Config file example:** `clamav.client_timeout`
  - **Type:** `float`
  - **Default:** `86400`

- **`ARCHIVEMATICA_WORKER_CLAMAV_CLIENT_BACKEND`**:
  - **Description:** the ClamAV backend used for anti-virus scanning. The two
    options that are available are: `clamscanner` (via CLI) and `clamdscanner`
    (over TCP or UNIX socket).
  - **Config file example:** `clamav.client_backend`
  - **Type:** `string`
  - **Default:** `clamdscanner`

- **`ARCHIVEMATICA_WORKER_CLAMAV_CLIENT_MAX_FILE_SIZE`**:
  - **Description:** files larger than this limit will not be scanned. The unit
    used is megabyte (MB).
  - **Config file example:** `clamav.client_max_file_size`
  - **Type:** `float`
  - **Default:** `2000`

- **`ARCHIVEMATICA_WORKER_CLAMAV_CLIENT_MAX_SCAN_SIZE`**:
  - **Description**: sets the maximum amount of data to be scanned for each
    input file. Files larger than this value will be scanned but only up to this
    limit. Archives and other containers are recursively extracted and scanned
    up to this value. The unit used is megabyte (MB).
  - **Config file example:** `clamav.client_max_scan_size`
  - **Type:** `float`
  - **Default:** `2000`

Database client:

- **`ARCHIVEMATICA_WORKER_DB_ENGINE`**
  - **Description:** a database setting. See [DATABASES] for more details.
  - **Config file example:** `db.engine`
  - **Type:** `string`
  - **Default:** `django.db.backends.mysql`

- **`ARCHIVEMATICA_WORKER_DB_DATABASE`**
  - **Description:** a database setting. See [DATABASES] for more details.
  - **Config file example:** `db.database`
  - **Type:** `string`
  - **Default:** `CCP`

- **`ARCHIVEMATICA_WORKER_DB_USER`**
  - **Description:** a database setting. See [DATABASES] for more details.
  - **Config file example:** `db.user`
  - **Type:** `string`
  - **Default:** `archivematica`

- **`ARCHIVEMATICA_WORKER_DB_PASSWORD`**
  - **Description:** a database setting. See [DATABASES] for more details.
  - **Config file example:** `db.password`
  - **Type:** `string`
  - **Default:** `demo`

- **`ARCHIVEMATICA_WORKER_DB_HOST`**
  - **Description:** a database setting. See [DATABASES] for more details.
  - **Config file example:** `db.host`
  - **Type:** `string`
  - **Default:** `localhost`

- **`ARCHIVEMATICA_WORKER_DB_PORT`**
  - **Description:** a database setting. See [DATABASES] for more details.
  - **Config file example:** `db.port`
  - **Type:** `string`
  - **Default:** `3306`

Email settings:

- **`ARCHIVEMATICA_WORKER_EMAIL_BACKEND`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.backend`
  - **Type:** `string`
  - **Default:** `django.core.mail.backends.console.EmailBackend`

- **`ARCHIVEMATICA_WORKER_EMAIL_HOST`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.host`
  - **Type:** `string`
  - **Default:** `smtp.gmail.com`

- **`ARCHIVEMATICA_WORKER_EMAIL_HOST_USER`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.host_user`
  - **Type:** `string`
  - **Default:** `your_email@example.com`

- **`ARCHIVEMATICA_WORKER_EMAIL_HOST_PASSWORD`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.host_password`
  - **Type:** `string`
  - **Default:** `None`

- **`ARCHIVEMATICA_WORKER_EMAIL_PORT`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.port`
  - **Type:** `integer`
  - **Default:** `587`

- **`ARCHIVEMATICA_WORKER_EMAIL_SSL_CERTFILE`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.ssl_certfile`
  - **Type:** `string`
  - **Default:** `None`

- **`ARCHIVEMATICA_WORKER_EMAIL_SSL_KEYFILE`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.ssl_keyfile`
  - **Type:** `string`
  - **Default:** `None`

- **`ARCHIVEMATICA_WORKER_EMAIL_USE_SSL`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.use_ssl`
  - **Type:** `boolean`
  - **Default:** `False`

- **`ARCHIVEMATICA_WORKER_EMAIL_USE_TLS`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.use_tls`
  - **Type:** `boolean`
  - **Default:** `True`

- **`ARCHIVEMATICA_WORKER_EMAIL_FILE_PATH`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.file_path`
  - **Type:** `string`
  - **Default:** `None`

- **`ARCHIVEMATICA_WORKER_EMAIL_DEFAULT_FROM_EMAIL`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.default_from_email`
  - **Type:** `string`
  - **Default:** `webmaster@example.com`

- **`ARCHIVEMATICA_WORKER_EMAIL_SUBJECT_PREFIX`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.subject_prefix`
  - **Type:** `string`
  - **Default:** `[Archivematica]`

- **`ARCHIVEMATICA_WORKER_EMAIL_TIMEOUT`**:
  - **Description:** an email setting. See [Sending email] for more details.
  - **Config file example:** `email.timeout`
  - **Type:** `integer`
  - **Default:** `300`

- **`ARCHIVEMATICA_WORKER_EMAIL_SERVER_EMAIL`**:
  - **Description:** an email setting. See [Sending email] for more details.
    When the value is `None`, Archivematica uses the value in `EMAIL_HOST_USER`.
  - **Config file example:** `email.server_email`
  - **Type:** `string`
  - **Default:** `None`

### Metadata XML validation variables

**This feature is experimental, please share your feedback!**

These variables specify how XML files contained in the `metadata` directory of a
SIP should be validated. Only applicable if
`ARCHIVEMATICA_WORKER_METADATA_XML_VALIDATION_ENABLED` is set.

- **`ARCHIVEMATICA_WORKER_METADATA_XML_VALIDATION_SETTINGS_FILE`**:
  - **Description:** Path to a Python module containing the following Django settings:
    - `XML_VALIDATION`: a dictionary which keys are strings that contain either
      an [XML schema location], an [XML namespace] or an XML element tag, and
      which values are either strings that contain an absolute local path or an
      external URL to an XML schema file, or `None` to indicate that no
      validation should be performed.
    - `XML_VALIDATION_FAIL_ON_ERROR`: a boolean that indicates if the SIP ingest
      workflow should stop on validation errors. Defaults to `False`.

      An `ImproperlyConfigured` exception will be raised if the Python module
      cannot be imported or does not contain the required settings.

      The goal of the validation process is to determine a **validation key**
      from the root node of each XML file in the `metadata` directory of the SIP
      and then get an XML schema file to validate against using the `XML_VALIDATION`
      dictionary. If the value for the validation key is set to `None` the XML
      metadata file is added to the METS without any validation.

      The validation key of each metadata XML file is determined from its root
      node in the following order:

        1. The XML schema location defined in the `xsi:noNamespaceSchemaLocation`
           attribute
        1. The XML schema location defined in the `xsi:schemaLocation` attribute
        1. Its XML namespace
        1. Its tag name

      XML validation works with `DTD`, `XSD` and `RELAX NG` schema types (`.dtd`,
      `.xsd`, and `.rng`).

      This is how an `XML_VALIDATION` dictionary would look like:

          XML_VALIDATION = {
              #
              # Local XML schema
              "http://www.lido-schema.org": "/etc/archivematica/xml-schemas/lido.xsd",
              #
              # External XML schema URL
              "http://www.loc.gov/MARC21/slim": "https://www.loc.gov/standards/marcxml/schema/MARC21slim.xsd",
              #
              # Tag name of the root node. Setting the value to `None` skips validation
              "metadata": None,
          }

  - **Type:** `string`
  - **Default:** ``

## Logging configuration

The worker will look in `/etc/archivematica` for a file called
`worker.logging.json`, and if found, this file will override the default
behaviour described above.

The [`worker.logging.json`](./worker.logging.json) file in this
directory provides an example that implements the logging behaviour preferred by
most users.

[SECRET_KEY]: https://docs.djangoproject.com/en/1.8/ref/settings/#secret-key
[DATABASES]: https://docs.djangoproject.com/en/1.8/ref/settings/#databases
[Sending email]: https://docs.djangoproject.com/en/1.8/topics/email/
[XML schema location]: https://www.w3.org/TR/xmlschema-1/#xsi_schemaLocation
[XML namespace]: https://www.w3.org/TR/xml-names/#sec-namespaces
