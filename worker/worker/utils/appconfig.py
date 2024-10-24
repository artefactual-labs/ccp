"""
Configuration helper.

Config. attributes are declared on those settings files and they can be defined
by a dictionary indicating the 'section', 'option' and 'type' to be parsed by
the Config class. They can also be defined by a list of the same type of
dictionary and, in that case, the first obtained value will be the one returned.
Alternatively, they can include the 'section' and a 'process_function' callback
where a specific parsing process can be defined. Those callbacks must accept the
current appconfig Config object and the section.
"""

import configparser
import functools
import os

from django.core.exceptions import ImproperlyConfigured


def fallback_option(fn):
    def wrapper(*args, **kwargs):
        fallback = kwargs.pop("fallback", None)
        try:
            return fn(*args, **kwargs)
        except (configparser.NoSectionError, configparser.NoOptionError):
            if fallback:
                return fallback
            raise

    return functools.wraps(fn)(wrapper)


class EnvConfigParser(configparser.RawConfigParser):
    """
    EnvConfigParser enables the user to provide configuration defaults using
    the string environment, e.g. given:

      - String environment prefix (prefix) = "ARCHIVEMATICA_WORKER"
      - Configuration section: "network"
      - Configuration option: "tls"

    This parser will first try to find the configuration value in the string
    environment matching one of the two following keys:

      - ARCHIVEMATICA_WORKER_NETWORK_TLS
      - ARCHIVEMATICA_WORKER_TLS

    If the variable is not set in the string environment the reader falls back
    on the main configuration backend.

    Additionally, the getters (get(), getint(), etc...) accept a new parameter
    (fallback) that returns the value given to it instead of an exception when
    the section or option trying to be match are undefined.
    """

    ENVVAR_SEPARATOR = "_"

    def __init__(self, defaults=None, env=None, prefix=""):
        self._environ = env or os.environ
        self._prefix = prefix.rstrip("_")
        kwargs = {}
        kwargs["inline_comment_prefixes"] = (";",)
        super().__init__(defaults, **kwargs)

    def _get_envvar(self, section, option):
        for key in (
            self.ENVVAR_SEPARATOR.join([self._prefix, section, option]).upper(),
            self.ENVVAR_SEPARATOR.join([self._prefix, option]).upper(),
        ):
            if key in self._environ:
                return self._environ[key]

    @fallback_option
    def get(self, section, option, **kwargs):
        ret = self._get_envvar(section, option)
        if ret:
            return ret
        return super().get(section, option, **kwargs)

    @fallback_option
    def getint(self, *args, **kwargs):
        return super().getint(*args, **kwargs)

    @fallback_option
    def getfloat(self, *args, **kwargs):
        return super().getfloat(*args, **kwargs)

    @fallback_option
    def getboolean(self, *args, **kwargs):
        return super().getboolean(*args, **kwargs)

    @fallback_option
    def getiboolean(self, *args, **kwargs):
        return not self.getboolean(*args, **kwargs)


class Config:
    """EnvConfigParser wrapper"""

    def __init__(self, env_prefix, attrs):
        self.config = EnvConfigParser(prefix=env_prefix)
        self.attrs = attrs

    INVALID_ATTR_MSG = (
        "Invalid attribute: %s. Make sure the entry in the"
        " attribute has all the fields needed (section, option,"
        " type)."
    )

    UNDEFINED_ATTR_MSG = "The following configuration attribute must be defined: %s."

    def read_defaults(self, fp):
        self.config.read_file(fp)

    def read_files(self, files):
        self.config.read(files)

    def get(self, attr, default=None):
        if attr not in self.attrs:
            raise ImproperlyConfigured(
                "Unknown attribute: %s. Make sure the "
                "attribute has been included in the "
                "attribute list." % attr
            )

        attr_opts = self.attrs[attr]
        if isinstance(attr_opts, list):
            return self.get_from_opts_list(attr, attr_opts, default=default)
        if all(k in attr_opts for k in ("section", "process_function")):
            return attr_opts["process_function"](self, attr_opts["section"])
        if not all(k in attr_opts for k in ("section", "option", "type")):
            raise ImproperlyConfigured(self.INVALID_ATTR_MSG % attr)

        getter = "get{}".format(
            "" if attr_opts["type"] == "string" else attr_opts["type"]
        )
        kwargs = {"section": attr_opts["section"], "option": attr_opts["option"]}
        if default is not None:
            kwargs["fallback"] = default
        elif "default" in attr_opts:
            kwargs["fallback"] = attr_opts["default"]

        try:
            return getattr(self.config, getter)(**kwargs)
        except (configparser.NoSectionError, configparser.NoOptionError):
            raise ImproperlyConfigured(self.UNDEFINED_ATTR_MSG % attr)

    def get_from_opts_list(self, attr, attr_opts_list, default=None):
        if not all(
            all(k in attr_opts for k in ("section", "option", "type"))
            for attr_opts in attr_opts_list
        ):
            raise ImproperlyConfigured(self.INVALID_ATTR_MSG % attr)
        for attr_opts in attr_opts_list:
            opt_type = attr_opts["type"]
            getter = "get{}".format({"string": ""}.get(opt_type, opt_type))
            kwargs = {"section": attr_opts["section"], "option": attr_opts["option"]}
            if default is not None:
                kwargs["fallback"] = default
            elif "default" in attr_opts:
                kwargs["fallback"] = attr_opts["default"]
            try:
                return getattr(self.config, getter)(**kwargs)
            except (
                configparser.NoSectionError,
                configparser.NoOptionError,
                ValueError,
            ):
                pass
        raise ImproperlyConfigured(self.UNDEFINED_ATTR_MSG % attr)
