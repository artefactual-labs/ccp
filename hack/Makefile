.DEFAULT_GOAL := help

# Paths for Docker named volumes
AM_PIPELINE_DATA ?= $(HOME)/.ccp/am-pipeline-data
SS_LOCATION_DATA ?= $(HOME)/.ccp/ss-location-data

CALLER_UID=$(shell id -u)
CALLER_GID=$(shell id -g)

SRCDIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/../src)

NULL :=
SPACE := $(NULL) $(NULL)
COMMA := ,

define compose_all
	docker compose -f docker-compose.yml -f docker-compose.acceptance-tests.yml -f docker-compose.pmm.yml $(1)
endef

define compose_amauat
	docker compose -f docker-compose.yml -f docker-compose.acceptance-tests.yml $(1)
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
	mkdir -p ${SS_LOCATION_DATA}
	docker volume create \
		--opt type=none \
		--opt o=bind \
		--opt device=$(SS_LOCATION_DATA) \
			ccp-ss-location-data
	# This workaround prevents Docker from creating this directory structure
	# inside the external SS_LOCATION_DATA volume directory assigning root as
	# the owner of some parts of it.
	mkdir -p $(SS_LOCATION_DATA)/archivematica/archivematica-sampledata

build:  # Build Compose services.
	docker compose build \
		--build-arg USER_ID=$(CALLER_UID) \
		--build-arg GROUP_ID=$(CALLER_GID)

bootstrap: bootstrap-storage-service bootstrap-db  ## Full bootstrap.

bootstrap-storage-service:  ## Bootstrap Storage Service (new database).
	$(call create_db,SS)
	docker compose run \
		--rm \
		--no-deps \
		--entrypoint /src/storage_service/manage.py \
			archivematica-storage-service \
				migrate --noinput
	docker compose run \
		--rm \
		--no-deps \
		--entrypoint /src/storage_service/manage.py \
			archivematica-storage-service \
				create_user \
					--username="test" \
					--password="test" \
					--email="test@test.com" \
					--api-key="test" \
					--superuser
	# SS needs to be restarted so the local space is created.
	# See #303 (https://git.io/vNKlM) for more details.
	docker compose restart --no-deps archivematica-storage-service

makemigrations-ss:
	docker compose run \
		--user $(CALLER_UID):$(CALLER_GID) \
		--rm \
		--no-deps \
		--entrypoint /src/storage_service/manage.py \
			archivematica-storage-service \
				makemigrations

manage:  ## Run Django /manage.py on Dashboard, suppling <command> [options] as value to ARG, e.g., `make manage-ss ARG=shell`
	docker compose run \
		--user $(CALLER_UID):$(CALLER_GID) \
		--rm \
		--no-deps \
		--entrypoint /src/src/MCPClient/lib/manage.py \
			archivematica-mcp-client \
				$(ARG)

manage-ss:  ## Run Django /manage.py on Storage Service, suppling <command> [options] as value to ARG, e.g., `make manage-ss ARG='shell --help'`
	docker compose run \
		--user $(CALLER_UID):$(CALLER_GID) \
		--rm \
		--no-deps \
		--entrypoint /src/storage_service/manage.py \
			archivematica-storage-service \
				$(ARG)

bootstrap-db:  ## Bootstrap Dashboard (new database).
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
					--ss-url="http://archivematica-storage-service:8000" \
					--ss-user="test" \
					--ss-api-key="test" \
					--site-url="http://archivematica-dashboard:8000"

restart-am-services:  ## Restart Archivematica services: MCPServer, MCPClient and Storage Service.
	docker compose restart --no-deps archivematica-mcp-client archivematica-storage-service

compile-requirements-am:  ## Run pip-compile for Archivematica
	docker compose run --workdir /src \
		-e XDG_CACHE_HOME=/tmp/pip-cache \
		--rm \
		--no-deps \
		--user $(CALLER_UID):$(CALLER_GID) \
		--entrypoint bash archivematica-mcp-client \
			-c "make pip-compile"

upgrade-requirements-am:  ## Run pip-upgrade for Archivematica
	docker compose run --workdir /src \
		-e XDG_CACHE_HOME=/tmp/pip-cache \
		--rm \
		--no-deps \
		--user $(CALLER_UID):$(CALLER_GID) \
		--entrypoint bash archivematica-mcp-client \
			-c "make pip-upgrade"

compile-requirements-ss:  ## Run pip-compile for Storage Service
	docker compose run --workdir /src \
		-e XDG_CACHE_HOME=/tmp/pip-cache \
		--rm \
		--no-deps \
		--user $(CALLER_UID):$(CALLER_GID) \
		--entrypoint bash archivematica-storage-service \
			-c "make pip-compile"

upgrade-requirements-ss:  ## Run pip-upgrade for Storage Service
	docker compose run --workdir /src \
		-e XDG_CACHE_HOME=/tmp/pip-cache \
		--rm \
		--no-deps \
		--user $(CALLER_UID):$(CALLER_GID) \
		--entrypoint bash archivematica-storage-service \
			-c "make pip-upgrade"

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

flush-search:  ## Delete Elasticsearch indices.
	docker compose exec archivematica-mcp-client curl -XDELETE "http://elasticsearch:9200/aips,aipfiles,transfers,transferfiles"

flush-logs:  ## Delete container logs - requires root privileges.
	@./helpers/flush-docker-logs.sh

# These include the database names used in the settings.testmysql
# modules plus the prefixed and suffixed variations used by Django and
# tox when running in parallel mode
__TEST_DBS_MCPCLIENT = MCPCLIENTTEST test_MCPCLIENTTEST test_MCPCLIENTTEST_mcp-client
__TEST_DBS_MIGRATIONS = DASHBOARDTEST SSTEST
__TEST_DBS_DASHBOARD = DASHBOARDTEST test_DASHBOARDTEST test_DASHBOARDTEST_dashboard test_DASHBOARDTEST_archivematica-common
__TEST_DBS_MCPSERVER = MCPSERVERTEST test_MCPSERVERTEST test_MCPSERVERTEST_mcp-server
__TEST_DBS_STORAGE_SERVICE = test_SSTEST test_SSTEST_storage-service
__TEST_DBS := $(__TEST_DBS_MCPCLIENT) $(__TEST_DBS_MIGRATIONS) $(__TEST_DBS_DASHBOARD) $(__TEST_DBS_MCPSERVER) $(__TEST_DBS_STORAGE_SERVICE) test_MCP test_SS
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

__TOXENVS_MCPSERVER := mcp-server
test-mcp-server: start-mysql  ## Run MCPServer tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_MCPSERVER))
	$(call run_toxenvs,$(__TOXENVS_MCPSERVER))

__TOXENVS_MCPCLIENT = mcp-client
test-mcp-client: start-mysql  ## Run MCPClient tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_MCPCLIENT))
	$(call run_toxenvs,$(__TOXENVS_MCPCLIENT))

__TOXENVS_DASHBOARD = dashboard
test-dashboard: start-mysql  ## Run Dashboard tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_DASHBOARD))
	$(call run_toxenvs,$(__TOXENVS_DASHBOARD))

__TOXENVS_STORAGE_SERVICE = storage-service
test-storage-service: start-mysql  ## Run Storage Service tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_STORAGE_SERVICE))
	$(call run_toxenvs,$(__TOXENVS_STORAGE_SERVICE))

test-storage-service-integration:  ## Run Storage Service unit and integration tests using MySQL and MinIO.
	$(CURDIR)/submodules/archivematica-storage-service/tests/integration/run.sh

__TOXENVS_ARCHIVEMATICA_COMMON = archivematica-common
test-archivematica-common: start-mysql  ## Run Archivematica Common tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_DASHBOARD))
	$(call run_toxenvs,$(__TOXENVS_ARCHIVEMATICA_COMMON))

__TOXENVS_MIGRATIONS = migrations-dashboard migrations-storage-service
test-migrations: start-mysql  ## Check there are no pending migrations.
	$(call create_dbs,$(__TEST_DBS_MIGRATIONS))
	$(call run_toxenvs,$(__TOXENVS_MIGRATIONS))

__TOXENVS_LINTING = linting
test-linting:  ## Check linting.
	$(call run_toxenvs,$(__TOXENVS_LINTING))

__TOXENVS_ALL := $(__TOXENVS_MCPSERVER) $(__TOXENVS_MCPCLIENT) $(__TOXENVS_DASHBOARD) $(__TOXENVS_STORAGE_SERVICE) $(__TOXENVS_ARCHIVEMATICA_COMMON)
test-all: start-mysql  ## Run all tests.
	$(call grant_all_on_dbs,$(__TEST_DBS_MCPSERVER) $(__TEST_DBS_MCPCLIENT) $(__TEST_DBS_DASHBOARD) $(__TEST_DBS_STORAGE_SERVICE) $(__TEST_DBS_ARCHIVEMATICA_COMMON))
	$(call run_toxenvs,$(__TOXENVS_ALL))

stop:  # Stop all containers.
	$(call compose_all, stop)

help:  ## Print this help message.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
