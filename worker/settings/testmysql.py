from .test import *

DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.mysql",
        "NAME": "WORKERTEST",
        "USER": "archivematica",
        "PASSWORD": "demo",
        "HOST": "mysql",
        "PORT": "3306",
        "CONN_MAX_AGE": 600,
        "TEST": {"NAME": "test_WORKERTEST"},
    }
}
