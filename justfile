set shell := ["bash", "-uc"]

[private]
default: _check_uv
  @just --list --unsorted

_check_uv:
  #!/usr/bin/env bash
  if ! command -v uv > /dev/null; then
    echo "uv is not installed. Please install uv to proceed."
    exit 1
  fi

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

# Run the development environment.
run:
  make -C hack run

# Submit a transfer in the dev environment using the Admin API.
transfer:
  ./hack/helpers/transfer-via-api.sh

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

# Show recent commits in upstream (qa/1.x).
git-log-recent-upstream:
  #!/usr/bin/env bash
  if ! git remote get-url upstream > /dev/null 2>&1; then
      git remote add -f upstream https://github.com/artefactual/archivematica.git
  else
      git fetch upstream
  fi
  git log --oneline upstream/qa/1.x ^HEAD

# Update worker dependencies.
worker-update-deps:
  #!/usr/bin/env bash
  cd worker
  uv sync --frozen
  uv lock --upgrade

# List outdated worker dependencies.
worker-list-outdated-deps:
  #!/usr/bin/env bash
  cd worker
  uv sync --frozen
  # TODO: https://github.com/astral-sh/uv/issues/2150
  # - uv has not implemented yet: `uv pip list --outdated`.
  # - this works but it's slow: `uv run --with=pip pip list --outdated`
  uv pip list --format=freeze | \
    sed 's/==.*//' | \
    uv pip compile - --no-deps --no-header | \
    uv pip compile - --no-deps --no-header | \
    diff <(uv pip list --format=freeze) - -y --suppress-common-lines || :

# Test worker migrations.
worker-test-migrations:
  #!/usr/bin/env bash
  cd worker
  uv sync --frozen
  uv run django-admin makemigrations --settings=settings.test --check --dry-run

# Test worker application.
worker-test-application *args:
  #!/usr/bin/env bash
  # TODO: missing test using mysql
  cd worker
  uv sync --frozen
  uv run pytest {{args}}

# Run pre-commit.
pre-commit *args:
  uvx pre-commit run --all-files {{args}}
