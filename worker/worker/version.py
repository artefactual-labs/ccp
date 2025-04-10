ARCHIVEMATICA_VERSION = (2, 0, 0)


def get_preservation_system_identifier():
    """Returns the system identifier including the application name."""
    return "Archivematica-%s" % get_version()


def get_full_version():
    return ".".join(str(x) for x in ARCHIVEMATICA_VERSION)
