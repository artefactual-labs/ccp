import uuid

from django.core.management.base import BaseCommand
from django.core.management.base import CommandError
from django.forms import Form
from django.forms import CharField
from django.forms import BooleanField
from django.utils.translation import gettext_lazy as _
from django.http import QueryDict
from django.utils import termcolors
import storageService as storage_service
from django.conf import settings as django_settings
from django.contrib.auth import get_user_model
from main.models import Agent
from main.models import DashboardSetting
from main.models import User
from tastypie.models import ApiKey


def create_super_user(username, email, password, key):
    UserModel = get_user_model()
    # Create the new super user if it doesn't already exist
    try:
        user = UserModel._default_manager.get(**{UserModel.USERNAME_FIELD: username})
    except UserModel.DoesNotExist:
        # User doesn't exist, create it
        user = UserModel._default_manager.db_manager("default").create_superuser(
            username, email, password
        )
    # Create or update the user's api key
    ApiKey.objects.update_or_create(user=user, defaults={"key": key})


def setup_pipeline(org_name, org_identifier, site_url):
    dashboard_uuid = get_setting("dashboard_uuid")
    # Setup pipeline only if dashboard_uuid doesn't already exists
    if dashboard_uuid:
        return

    # Assign UUID to Dashboard
    dashboard_uuid = str(uuid.uuid4())
    set_setting("dashboard_uuid", dashboard_uuid)

    if org_name != "" or org_identifier != "":
        agent = Agent.objects.default_organization_agent()
        agent.name = org_name
        agent.identifiertype = "repository code"
        agent.identifiervalue = org_identifier
        agent.save()

    if site_url:
        set_setting("site_url", site_url)


def setup_pipeline_in_ss(use_default_config=False):
    # Check if pipeline is already registered on SS
    dashboard_uuid = get_setting("dashboard_uuid")
    try:
        storage_service.get_pipeline(dashboard_uuid)
    except Exception:
        print("SS inaccessible or pipeline not registered.")
    else:
        # If pipeline is already registered on SS, then exit
        print("This pipeline is already configured on SS.")
        return

    if not use_default_config:
        # Storage service manually set up, just register Pipeline if
        # possible. Do not provide additional information about the shared
        # path, or API, as this is probably being set up in the storage
        # service manually.
        storage_service.create_pipeline()
        return

    # Post first user & API key
    user = User.objects.all()[0]
    api_key = ApiKey.objects.get(user=user)

    # Retrieve remote name
    try:
        setting = DashboardSetting.objects.get(name="site_url")
    except DashboardSetting.DoesNotExist:
        remote_name = None
    else:
        remote_name = setting.value

    # Create pipeline, tell it to use default setup
    storage_service.create_pipeline(
        create_default_locations=True,
        shared_path=django_settings.SHARED_DIRECTORY,
        remote_name=remote_name,
        api_username=user.username,
        api_key=api_key.key,
    )


class Command(BaseCommand):
    def add_arguments(self, parser):
        parser.add_argument("--username", required=True)
        parser.add_argument("--email", required=True)
        parser.add_argument("--password", required=True)
        parser.add_argument("--api-key", required=True)
        parser.add_argument("--org-name", required=True)
        parser.add_argument("--org-id", required=True)
        parser.add_argument("--ss-url", required=True)
        parser.add_argument("--ss-user", required=True)
        parser.add_argument("--ss-api-key", required=True)
        parser.add_argument(
            "--whitelist",
            required=False,
            help="Deprecated. Please use --allowlist instead.",
        )
        parser.add_argument("--allowlist", required=False)
        parser.add_argument("--site-url", required=False)

    def save_ss_settings(self, options):
        POST = QueryDict("", mutable=True)
        POST.update(
            {
                "storage_service_url": options["ss_url"],
                "storage_service_user": options["ss_user"],
                "storage_service_apikey": options["ss_api_key"],
            }
        )
        form = StorageSettingsForm(POST)
        if not form.is_valid():
            raise CommandError("SS attributes are invalid")
        form.save()

    def handle(self, *args, **options):
        # Not needed in Django 1.9+.
        self.style.SUCCESS = termcolors.make_style(opts=("bold",), fg="green")

        setup_pipeline(options["org_name"], options["org_id"], options["site_url"])
        create_super_user(
            options["username"],
            options["email"],
            options["password"],
            options["api_key"],
        )
        self.save_ss_settings(options)
        setup_pipeline_in_ss(use_default_config=True)
        set_api_allowlist(options["whitelist"] or options["allowlist"])
        self.stdout.write(self.style.SUCCESS("Done!\n"))


def get_setting(setting, default=""):
    try:
        setting = DashboardSetting.objects.get(name=setting)
        return setting.value
    except Exception:
        return default


def set_setting(setting, value=""):
    try:
        setting_data = DashboardSetting.objects.get(name=setting)
    except Exception:
        setting_data = DashboardSetting.objects.create()
        setting_data.name = setting
    # ``DashboardSetting.value`` is a string-based field. The empty string is
    # the way to represent the lack of data, therefore NULL values are avoided.
    if value is None:
        value = ""
    setting_data.value = value
    setting_data.save()


def set_api_allowlist(allowlist):
    """Set API allowlist setting.

    ``allowlist`` (str) is a space-separated list of IP addresses with access
    to the public API. If falsy, all clients are allowed.
    """
    if not allowlist:
        allowlist = ""
    return set_setting("api_whitelist", allowlist)


class SettingsForm(Form):
    """Base class form to save settings to DashboardSettings."""

    def save(self, *args, **kwargs):
        """Save all the form fields to the DashboardSettings table."""
        for key in self.cleaned_data:
            # Save the value
            set_setting(key, self.cleaned_data[key])


class StorageSettingsForm(SettingsForm):
    class StripCharField(CharField):
        """
        Strip the value of leading and trailing whitespace.

        This won't be needed in Django 1.9, see
        https://docs.djangoproject.com/en/1.9/ref/forms/fields/#django.forms.CharField.strip.
        """

        def to_python(self, value):
            return super(CharField, self).to_python(value).strip()

    storage_service_url = CharField(
        label=_("Storage Service URL"),
        help_text=_(
            "Full URL of the storage service. E.g. https://192.168.168.192:8000"
        ),
    )
    storage_service_user = CharField(
        label=_("Storage Service User"),
        help_text=_("User in the storage service to authenticate as. E.g. test"),
    )
    storage_service_apikey = StripCharField(
        label=_("API key"),
        help_text=_(
            "API key of the storage service user. E.g. 45f7684483044809b2de045ba59dc876b11b9810"
        ),
    )
    storage_service_use_default_config = BooleanField(
        required=False,
        initial=True,
        label=_("Use default configuration"),
        help_text=_(
            "You have to manually set up a custom configuration if the default configuration is not selected."
        ),
    )
