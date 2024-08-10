.DEFAULT_GOAL := help

# Paths for Docker named volumes
AM_PIPELINE_DATA ?= $(HOME)/.ccp/am-pipeline-data

CALLER_UID=$(shell id -u)
CALLER_GID=$(shell id -g)

SRCDIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/../src)

NULL :=
SPACE := $(NULL) $(NULL)
COMMA := ,

define compose_all
	docker compose -f docker-compose.yml -f docker-compose.acceptance-tests.yml -f docker-compose.pmm.yml $(1)
endef

define compose_tests
	docker compose -f docker-compose.yml -f docker-compose.tests.yml $(1)
endef

define run_toxenvs
	$(call compose_tests, \
		run \
			-e DJANGO_SETTINGS_MODULE=settings.testmysql \
			-e PYTEST_ADDOPTS=$(subst $(SPACE),\$(SPACE),${PYTEST_ADDOPTS}) \
			--user $(CALLER_UID):$(CALLER_GID) \
			--workdir /src \
			--rm \
			--entrypoint tox \
				archivematica-tests \
					-e $(subst $(SPACE),$(COMMA),$(strip $(1))) \
					${TOXARGS})
endef

start-mysql: test-build
	$(call compose_tests, \
		run \
			--user $(CALLER_UID):$(CALLER_GID) \
			--rm \
			--entrypoint bash \
				archivematica-tests \
					/src/hack/wait-for-it.sh mysql:3306 --timeout=30)

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

define drop_db
	docker compose exec -T mysql mysql -hlocalhost -uroot -p12345 -e 'DROP DATABASE IF EXISTS `$(1)`;'
endef

define create_dbs
	$(foreach test_db,$(1),$(call create_db,$(test_db));)
endef

define grant_all_on_dbs
	$(foreach test_db,$(1),$(call grant_all_on_db,$(test_db));)
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

iterate:
	$(MAKE) flush-shared-dir
	docker compose up -d --force-recreate --build --no-deps archivematica-ccp archivematica-mcp-client
	docker compose logs -f archivematica-ccp archivematica-mcp-client

manage:  ## Run Django /manage.py on Dashboard, suppling <command> [options] as value to ARG, e.g., `make manage-ss ARG=shell`
	docker compose run \
		--user $(CALLER_UID):$(CALLER_GID) \
		--rm \
		--no-deps \
		--entrypoint /src/src/MCPClient/lib/manage.py \
			archivematica-mcp-client \
				$(ARG)

bootstrap:  ## Bootstrap MCP (new database).
	$(call create_db,MCP)
	docker compose run \
		--rm \
		--no-deps \
		--entrypoint /src/src/MCPClient/lib/manage.py \
			archivematica-mcp-client \
				migrate --noinput
	docker compose run \
		--rm \
		--no-deps \
		--entrypoint /src/src/MCPClient/lib/manage.py \
			archivematica-mcp-client \
				install \
					--username="test" \
					--password="test" \
					--email="test@test.com" \
					--org-name="test" \
					--org-id="test" \
					--api-key="test" \
					--site-url="http://archivematica-dashboard:8000"

restart-am-services:  ## Restart Archivematica services.
	docker compose restart --no-deps archivematica-mcp-client

compile-requirements:  ## Run pip-compile for Archivematica
	docker compose run --workdir /src \
		-e XDG_CACHE_HOME=/tmp/pip-cache \
		--rm \
		--no-deps \
		--user $(CALLER_UID):$(CALLER_GID) \
		--entrypoint bash archivematica-mcp-client \
			-c "pip-compile --allow-unsafe --output-file requirements.txt requirements.in && pip-compile --allow-unsafe --output-file requirements-dev.txt requirements-dev.in"

upgrade-requirements:  ## Run pip-upgrade for Archivematica
	docker compose run --workdir /src \
		-e XDG_CACHE_HOME=/tmp/pip-cache \
		--rm \
		--no-deps \
		--user $(CALLER_UID):$(CALLER_GID) \
		--entrypoint bash archivematica-mcp-client \
			-c "pip-compile --allow-unsafe --upgrade --output-file requirements.txt requirements.in && pip-compile --allow-unsafe --upgrade --output-file requirements-dev.txt requirements-dev.in"

db:  ## Connect to the MySQL server using the CLI.
	docker compose exec mysql mysql -hlocalhost -uroot -p12345

dbdump:  ## Dump database.
	@docker-compose exec mysql mysqldump -hlocalhost -uroot -p12345 MCP

dbschemadump:  ## Dump shema.
	@docker-compose exec mysql mysqldump -hlocalhost -uroot -p12345 --no-data --skip-add-drop-table MCP

flush: flush-shared-dir bootstrap restart-am-services  ## Delete ALL user data.

flush-shared-dir-mcp-configs:  ## Delete processing configurations - it restarts MCPServer.
	rm -f ${AM_PIPELINE_DATA}/sharedMicroServiceTasksConfigs/processingMCPConfigs/defaultProcessingMCP.xml
	rm -f ${AM_PIPELINE_DATA}/sharedMicroServiceTasksConfigs/processingMCPConfigs/automatedProcessingMCP.xml
	docker compose restart --no-deps archivematica-ccp

flush-shared-dir:  ## Delete contents of the shared directory data volume.
	rm -rf ${AM_PIPELINE_DATA}/*

flush-logs:  ## Delete container logs - requires root privileges.
	@./helpers/flush-docker-logs.sh

# These include the database names used in the settings.testmysql
# modules plus the prefixed and suffixed variations used by Django and
# tox when running in parallel mode
__TEST_DBS_MCPCLIENT = MCPCLIENTTEST test_MCPCLIENTTEST test_MCPCLIENTTEST_mcp-client
__TEST_DBS_MIGRATIONS = MCPCLIENTTEST
__TEST_DBS := $(__TEST_DBS_MCPCLIENT) $(__TEST_DBS_MIGRATIONS) test_MCP
flush-test-dbs: start-mysql
	$(foreach test_db,$(__TEST_DBS),$(call drop_db,$(test_db));)

test-build:  ## Build archivematica-tests image.
	docker build \
		-t archivematica-tests \
		-f $(CURDIR)/Dockerfile \
		--build-arg TARGET=archivematica-tests \
		--build-arg UBUNTU_VERSION \
		--build-arg PYTHON_VERSION \
			../

__TOXENVS_MCPCLIENT = mcp-client
test-mcp-client: start-mysql  ## Run MCPClient tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_MCPCLIENT))
	$(call run_toxenvs,$(__TOXENVS_MCPCLIENT))

__TOXENVS_ARCHIVEMATICA_COMMON = archivematica-common
test-archivematica-common: start-mysql  ## Run Archivematica Common tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_MCPCLIENT))
	$(call run_toxenvs,$(__TOXENVS_ARCHIVEMATICA_COMMON))

__TOXENVS_MIGRATIONS = migrations
test-migrations: start-mysql  ## Check there are no pending migrations.
	$(call create_dbs,$(__TEST_DBS_MIGRATIONS))
	$(call run_toxenvs,$(__TOXENVS_MIGRATIONS))

__TOXENVS_LINTING = linting
test-linting:  ## Check linting.
	$(call run_toxenvs,$(__TOXENVS_LINTING))

__TOXENVS_ALL := $(__TOXENVS_MCPCLIENT) $(__TOXENVS_ARCHIVEMATICA_COMMON)
test-all: start-mysql  ## Run all tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_MCPCLIENT) $(__TEST_DBS_ARCHIVEMATICA_COMMON))
	$(call run_toxenvs,$(__TOXENVS_ALL))

stop:  # Stop all containers.
	$(call compose_all, stop)

help:  ## Print this help message.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
