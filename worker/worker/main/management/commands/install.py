import uuid
from argparse import BooleanOptionalAction

from django.contrib.auth import get_user_model
from django.core.management import call_command
from django.core.management.base import BaseCommand
from tastypie.models import ApiKey

from worker.main.models import Agent
from worker.main.models import DashboardSetting


class Command(BaseCommand):
    def add_arguments(self, parser):
        parser.add_argument(
            "--migrate",
            default=False,
            action=BooleanOptionalAction,
            help="Apply migrations.",
        )
        parser.add_argument("--username", required=True)
        parser.add_argument("--email", required=True)
        parser.add_argument("--password", required=True)
        parser.add_argument("--api-key", required=True)
        parser.add_argument("--org-name", required=True)
        parser.add_argument("--org-id", required=True)
        parser.add_argument(
            "--whitelist",
            required=False,
            help="Deprecated. Please use --allowlist instead.",
        )
        parser.add_argument("--allowlist", required=False)
        parser.add_argument("--site-url", required=False)

    def handle(self, *args, **options):
        if options["migrate"] is True:
            self.print_banner("Apply migrations")
            call_command("migrate", interactive=False)
            self.print_done()

        self.print_banner("Set up pipeline")
        setup_pipeline(options["org_name"], options["org_id"], options["site_url"])
        self.print_done()

        self.print_banner("Create superuser")
        create_super_user(
            options["username"],
            options["email"],
            options["password"],
            options["api_key"],
        )
        self.print_done()

        self.print_banner("Set up API allowlist")
        set_api_allowlist(options["whitelist"] or options["allowlist"])
        self.print_done()

    def print_done(self):
        self.stdout.write(self.style.SUCCESS("Done!\n"))

    def print_banner(self, text):
        length = len(text)
        border = "+" + "-" * (length + 2) + "+"
        middle = f"+ {text} +"
        self.stdout.write(f"\n{border}\n{middle}\n{border}\n")


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
