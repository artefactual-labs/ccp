#!/usr/bin/env -S just --justfile

set shell := ["bash", "-uc"]

bine_path := `go tool bine path`
export PATH := bine_path + ":" + env_var("PATH")

import 'hack/dev.just'

[private]
default:
  @just --list --unsorted

[private]
install tool:
  @echo "Installing {{ tool }}..."
  @go tool bine get {{ tool }} 1> /dev/null

[private]
filtered-packages:
  @go list ./... | grep -Ev \
    "github.com/artefactual-labs/ccp/internal/api/gen/.*|"\
    "github.com/artefactual-labs/ccp/internal/.*/enums|"\
    "github.com/artefactual-labs/ccp/internal/store/sqlcmysql|"\
    "github.com/artefactual-labs/ccp/internal/.*mock"

# Run the generate-dumps Dagger pipeline.
e2e-dump:
  dagger call --progress=plain generate-dumps export --path=e2e/testdata/dumps

# Run the e2e Dagger pipeline.
e2e:
  dagger call --progress=plain etoe

# Launch amflow.
amflow:
  amflow edit --file ./internal/workflow/assets/workflow.json

# Launch grpcui.
grpcui:
  grpcui -plaintext -H "Authorization: ApiKey test:test" localhost:63030

# Tag and release new version.
release:
    #!/usr/bin/env bash
    set -euo pipefail
    branch=qa/2.x
    git checkout ${branch} > /dev/null 2>&1
    git diff-index --quiet HEAD || (echo "Git directory is dirty" && exit 1)
    version=v$(semver bump prerelease beta.. $(git describe --abbrev=0))
    echo "Detected version: ${version}"
    read -n 1 -p "Is that correct (y/N)? " answer
    echo
    case ${answer:0:1} in
        y|Y )
            echo "Tagging release with version ${version}"
        ;;
        * )
            echo "Aborting"
            exit 1
        ;;
    esac
    git tag -m "Release ${version}" $version
    git push origin refs/tags/$version

# Run pre-commit.
pre-commit *args:
  uvx pre-commit run --all-files {{args}}

# List outdated dependencies.
list-go-deps *args="-update -direct": (install "go-mod-outdated")
  go list -u -m -json all | go-mod-outdated {{args}}

# Format protobuf files with buf.
buf-format: (install "buf")
  buf format --write

# Check protobuf files with buf.
buf-checks: (install "buf")
  buf format --diff --exit-code
  buf lint

# Generate all code assets.
gen: gen-mocks gen-enums gen-sqlc gen-buf gen-web

# Generate mocks.
gen-mocks: (install "mockgen")
  mockgen -typed -source=./internal/store/store.go -destination=./internal/store/storemock/mock_store.go -package=storemock Store
  mockgen -typed -source=./internal/controller/dispatcher/dispatcher.go -destination=./internal/controller/dispatcher/dispatchermock/dispatcher_mock.go -package=dispatchermock Backend
  mockgen -typed -source=./internal/controller/dispatcher/workhub.go -destination=./internal/controller/dispatcher/dispatchermock/workhub_mock.go -package=dispatchermock workhubSubmitter

# Generate enums.
gen-enums: (install "go-enum")
	go-enum \
		-f internal/store/enums/job_status.go \
		-f internal/store/enums/package_status.go \
		-f internal/store/enums/package_type.go \
		--marshal --nocase --names --ptr --flag --sql

# Generate sqlc code.
[working-directory: 'internal/store/sqlc']
gen-sqlc: (install "sqlc")
  sqlc generate

# Generate buf code.
gen-buf: (install "buf")
  buf generate

# Genereate web build.
gen-web:
  npm --prefix={{justfile_directory()}}/web run build

# Run all tests with gotestsum.
test args="" goargs="": (install "gotestsum")
  gotestsum --format=testdox {{ args }} $(just filtered-packages) -- {{ goargs }}

# Run all tests and output a coverage report using tparse.
tparse: (install "tparse")
  go test -count=1 -json -cover $(just filtered-packages) | tparse -follow -all -notests

# Lint the code.
lint *args="--fix": (install "golangci-lint")
  golangci-lint run {{ args }}

# Format the code.
fmt *args: (install "golangci-lint")
  golangci-lint fmt {{ args }}

# Show recent commits in upstream (qa/1.x).
[group("worker")]
worker-upstream-changes:
  #!/usr/bin/env bash
  if ! git remote get-url upstream > /dev/null 2>&1; then
      git remote add -f upstream https://github.com/artefactual/archivematica.git
  else
      git fetch upstream
  fi
  git log --oneline upstream/qa/1.x ^HEAD

# Update worker dependencies.
[group("worker")]
worker-update-deps: (install "uv")
  #!/usr/bin/env bash
  cd worker
  uv sync --frozen
  uv lock --upgrade

# List outdated worker dependencies.
[group("worker")]
worker-list-outdated-deps: (install "uv")
  uv --project=worker pip list --outdated

# Test worker migrations.
[group("worker")]
worker-test-migrations: (install "uv")
  uv run --project=worker django-admin makemigrations --settings=settings.test --check --dry-run

# Test worker application.
[group("worker")]
worker-test-application *args: (install "uv")
  uv run --project=worker pytest {{args}}
