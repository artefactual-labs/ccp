import os

if __name__ == "__main__":
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "settings.common")

    from worker.client.mcp import main

    main()
