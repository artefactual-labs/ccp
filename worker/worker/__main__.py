import os


def main():
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "settings.common")

    from worker.client.mcp import main as run

    run()


if __name__ == "__main__":
    main()
