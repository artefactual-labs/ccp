.DEFAULT_GOAL := help

# Paths for Docker named volumes
AM_PIPELINE_DATA ?= $(HOME)/.ccp/am-pipeline-data

CALLER_UID=$(shell id -u)
CALLER_GID=$(shell id -g)

NULL :=
SPACE := $(NULL) $(NULL)
COMMA := ,

define create_db
	docker compose exec -T mysql mysql -hlocalhost -uroot -p12345 -e '\
		DROP DATABASE IF EXISTS `$(1)`; \
		CREATE DATABASE `$(1)`;'
	$(call grant_all_on_db,$(1))
endef

define grant_all_on_db
	docker compose exec -T mysql mysql -hlocalhost -uroot -p12345 -e '\
		GRANT ALL ON `$(1)`.* TO "archivematica"@"%";'
endef

create-volumes:  ## Create external data volumes.
	mkdir -p ${AM_PIPELINE_DATA}
	docker volume create \
		--opt type=none \
		--opt o=bind \
		--opt device=$(AM_PIPELINE_DATA) \
			ccp-am-pipeline-data

build:  # Build Compose services.
	docker compose build \
		--build-arg USER_ID=$(CALLER_UID) \
		--build-arg GROUP_ID=$(CALLER_GID)

run:
	$(MAKE) flush-shared-dir
	docker compose up -d --force-recreate --build --no-deps mysql ccp worker
	docker compose logs -f ccp worker

manage:  ## Run Django /manage.py on Dashboard, suppling <command> [options] as value to ARG, e.g., `make manage-ss ARG=shell`
	docker compose run \
		--user $(CALLER_UID):$(CALLER_GID) \
		--rm \
		--no-deps \
		--entrypoint /src/manage.py \
			worker \
				$(ARG)

bootstrap:  ## Bootstrap the database.
	$(call create_db,CCP)
	docker compose run \
		--rm \
		--no-deps \
		--entrypoint /src/manage.py \
			worker \
				install \
					--wait-for-db \
					--migrate \
					--username="test" \
					--password="test" \
					--email="test@test.com" \
					--org-name="test" \
					--org-id="test" \
					--api-key="test" \
					--site-url="http://archivematica-dashboard:8000"

restart:  ## Restart Archivematica services.
	docker compose restart --no-deps ccp worker

db:  ## Connect to the MySQL server using the CLI.
	docker compose exec mysql mysql -hlocalhost -uroot -p12345

dbdump:  ## Dump database.
	@docker-compose exec mysql mysqldump -hlocalhost -uroot -p12345 CCP

dbschemadump:  ## Dump shema.
	@docker-compose exec mysql mysqldump -hlocalhost -uroot -p12345 --no-data --skip-add-drop-table CCP

flush: flush-shared-dir bootstrap restart  ## Delete ALL user data.

flush-shared-dir-mcp-configs:  ## Delete processing configurations - it restarts the server.
	rm -f ${AM_PIPELINE_DATA}/sharedMicroServiceTasksConfigs/processingMCPConfigs/defaultProcessingMCP.xml
	rm -f ${AM_PIPELINE_DATA}/sharedMicroServiceTasksConfigs/processingMCPConfigs/automatedProcessingMCP.xml
	docker compose restart --no-deps ccp

flush-shared-dir:  ## Delete contents of the shared directory data volume.
	rm -rf ${AM_PIPELINE_DATA}/*

flush-logs:  ## Delete container logs - requires root privileges.
	@./helpers/flush-docker-logs.sh

stop:  # Stop all containers.
	docker compose stop

help:  ## Print this help message.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
