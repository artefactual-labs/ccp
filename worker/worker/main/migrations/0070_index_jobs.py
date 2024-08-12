import uuid

from django.db import migrations

from worker.main.models import UUIDField


class Migration(migrations.Migration):
    dependencies = [("main", "0069_remove_atk")]

    operations = [
        migrations.AlterField(
            model_name="job",
            name="sipuuid",
            field=UUIDField(
                max_length=36, db_column="SIPUUID", db_index=True, default=uuid.uuid4
            ),
        ),
        migrations.AlterIndexTogether(
            name="job",
            index_together={
                ("sipuuid", "createdtime", "createdtimedec"),
                (
                    "sipuuid",
                    "currentstep",
                    "microservicegroup",
                    "microservicechainlink",
                ),
                ("sipuuid", "jobtype", "createdtime", "createdtimedec"),
                ("jobtype", "currentstep"),
            },
        ),
    ]
