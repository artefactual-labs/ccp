COMPOSE_DIR := justfile_directory() / "hack"

[private]
@compose-submodules:
  git submodule update --init --recursive

[private]
@compose-build:
  echo "Building services..."
  docker compose --project-directory={{ COMPOSE_DIR }} build

[private]
[group("compose")]
@compose-flush:
  docker compose --project-directory={{ COMPOSE_DIR }} exec -T mysql mysql -hlocalhost -uroot -p12345 -e 'DROP DATABASE IF EXISTS `$(1)`; CREATE DATABASE `$(1)`;'

[private]
@compose-up-fg:
  echo "Starting containers..."
  docker compose --project-directory={{ COMPOSE_DIR }} up --menu --force-recreate mysql ccp worker

[private]
@compose-up-bg:
  echo "Starting containers..."
  docker compose --project-directory={{ COMPOSE_DIR }} up --detach --menu --force-recreate mysql ccp worker

# Run the Django management command.
[group("compose")]
@compose-manage *args:
  docker compose --project-directory={{ COMPOSE_DIR }} run --rm --no-deps --entrypoint /src/manage.py worker {{ args }}

# Run the development environment in the foreground.
[group("compose")]
compose-fg: compose-submodules compose-build compose-up-fg

# Run the development environment in the background.
[group("compose")]
compose-bg: compose-submodules compose-build compose-up-bg

# Submit a transfer via the API.
[group("compose")]
@compose-transfer-via-api:
  ./hack/helpers/transfer-via-api.sh

# Submit a transfer via the watched directory.
[group("compose")]
@compose-transfer-via-watched-dir:
  ./hack/helpers/transfer-via-watched-dir.sh

# Open the MySQL shell.
[group("compose")]
@compose-db:
  docker compose --project-directory={{ COMPOSE_DIR }} exec mysql mysql -hlocalhost -uroot -p12345
