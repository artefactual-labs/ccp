
PROJECT := ccp
MAKEDIR := hack/make
SHELL   := /bin/bash

.DEFAULT_GOAL := help
.PHONY: *

DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif

include hack/make/bootstrap.mk
include hack/make/dep_buf.mk
include hack/make/dep_goenum.mk
include hack/make/dep_golangci_lint.mk
include hack/make/dep_gotestsum.mk
include hack/make/dep_mockgen.mk
include hack/make/dep_oapi_codegen.mk
include hack/make/dep_sqlc.mk
include hack/make/dep_tparse.mk
include hack/make/enums.mk

# Lazy-evaluated list of tools.
TOOLS = $(BUF) \
	$(GOENUM) \
	$(GOLANGCI_LINT) \
	$(GOTESTSUM) \
	$(MOCKGEN) \
	$(OAPI_CODEGEN) \
	$(SQLC) \
	$(TPARSE)

define NEWLINE


endef

IGNORED_PACKAGES := \
	github.com/artefactual/archivematica/hack/ccp/internal/api/gen/% \
	github.com/artefactual/archivematica/hack/ccp/internal/shim/gen \
	github.com/artefactual/archivematica/hack/ccp/internal/%/enums \
	github.com/artefactual/archivematica/hack/ccp/internal/ssclient/ssclientmock \
	github.com/artefactual/archivematica/hack/ccp/internal/store/sqlcmysql \
	github.com/artefactual/archivematica/hack/ccp/internal/store/storemock
PACKAGES := $(shell go list ./...)
TEST_PACKAGES := $(filter-out $(IGNORED_PACKAGES),$(PACKAGES))
TEST_IGNORED_PACKAGES := $(filter $(IGNORED_PACKAGES),$(PACKAGES))

tools: # @HELP Install tools.
tools: $(TOOLS)

env: # @HELP Print Go env variables.
env:
	go env

tparse: # @HELP Run all tests and output a coverage report using tparse.
tparse: $(TPARSE)
	go test -count=1 -json -cover $(TEST_PACKAGES) | tparse -follow -all -notests

db: # @HELP Connect to the MySQL server using the CLI.
	docker compose exec mysql mysql -hlocalhost -uroot -p12345 MCP

test: # @HELP Run all tests and output a summary using gotestsum.
test: TFORMAT ?= short
test: GOTEST_FLAGS ?=
test: COMBINED_FLAGS ?= $(GOTEST_FLAGS) $(TEST_PACKAGES)
test: $(GOTESTSUM)
	gotestsum --format=$(TFORMAT) -- $(COMBINED_FLAGS)

test-race: # @HELP Run all tests with the race detector.
test-race:
	$(MAKE) test GOTEST_FLAGS="-race"

test-ci: # @HELP Run all tests in CI with coverage and the race detector.
test-ci:
	$(MAKE) test GOTEST_FLAGS="-race -coverprofile=covreport -covermode=atomic"

list-tested-packages: # @HELP Print a list of packages being tested.
list-tested-packages:
	$(foreach PACKAGE,$(TEST_PACKAGES),@echo $(PACKAGE)$(NEWLINE))

list-ignored-packages: # @HELP Print a list of packages ignored in testing.
list-ignored-packages:
	$(foreach PACKAGE,$(TEST_IGNORED_PACKAGES),@echo $(PACKAGE)$(NEWLINE))

lint: # @HELP Lint the project Go files with golangci-lint.
lint: OUT_FORMAT ?= colored-line-number
lint: LINT_FLAGS ?= --timeout=5m --fix
lint: $(GOLANGCI_LINT)
	golangci-lint run --out-format $(OUT_FORMAT) $(LINT_FLAGS)

.PHONY: confirm-deps
confirm-deps:
	@echo -n "Do you want to update? [y/N] " && read ans && [ $${ans:-N} = y ]

.PHONY: deps
deps: DIRECT_DEPENDENCIES := $(shell go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
deps:
	@clear
	@echo Direct dependencies:
	$(foreach DEP,$(DIRECT_DEPENDENCIES),@echo » $(DEP)$(NEWLINE))
	@echo
	@echo Available updates:
	@go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}» {{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null
	@echo
	@$(MAKE) confirm-deps
	$(foreach DEP,$(DIRECT_DEPENDENCIES),@go get $(DEP)$(NEWLINE))
	@go mod tidy

gen: # @HELP Generage code.
gen: gen-mocks gen-sqlc gen-enums gen-buf gen-web gen-shim

gen-mocks: # @HELP Generate mocks.
gen-mocks: $(MOCKGEN)
	mockgen -typed -source=./internal/store/store.go -destination=./internal/store/storemock/mock_store.go -package=storemock Store
	mockgen -typed -source=./internal/ssclient/ssclient.go -destination=./internal/ssclient/ssclientmock/mock_ssclient.go -package=ssclientmock Client

gen-sqlc: # @HELP Generate sqlc code.
gen-sqlc: $(SQLC)
	sqlc generate --file=$(CURDIR)/sqlc/sqlc.yaml

gen-enums: # @HELP Generate enums.
gen-enums: $(ENUMS)

gen-buf: # @HELP Generate buf.build assets.
gen-buf: $(BUF)
	buf generate --include-imports
	rm -rf $(CURDIR)/internal/api/gen/buf/ $(CURDIR)/web/src/gen/buf
	sed -i \
		-e 's/^from archivematica\.ccp\.admin\.v1beta1/from client\.gen\.archivematica\.ccp\.admin\.v1beta1/' \
		-e 's/^from buf\.validate/from client\.gen\.buf\.validate/' \
		$(CURDIR)/../../src/dashboard/src/client/gen/archivematica/ccp/admin/v1beta1/*.py \
		$(CURDIR)/../../src/dashboard/src/client/gen/buf/validate/*.py

gen-web: # @HELP Generate webui assets.
gen-web:
	npm --prefix=$(CURDIR)/web run build

gen-shim: # @HELP Generate Go server boilerplate from OpenAPIv3 spec.
gen-shim: SPEC_COMMIT := b0b9af20
gen-shim: SPEC_URL := https://raw.githubusercontent.com/artefactual-labs/archivematica-api-specification/$(SPEC_COMMIT)/typespec/tsp-output/@typespec/openapi3/openapi.v1.yaml
gen-shim: $(OAPI_CODEGEN)
	@echo "Downloading Archivematica API specification ($(SPEC_COMMIT))..."
	@echo "Full URL: $(SPEC_URL)"
	curl -Ls -o /tmp/amspec.yml $(SPEC_URL)
	cd $(CURDIR)/internal/shim && oapi-codegen -config=config.yml /tmp/amspec.yml

push: # @HELP Push new build of CCP to development environment.
push:
	docker compose up -d --force-recreate --build --no-deps archivematica-ccp
	docker compose restart --no-deps archivematica-mcp-client archivematica-dashboard

help: # @HELP Print this message.
help:
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' Makefile             \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-30s %s\n", $$1, $$2 };  \
	    '
