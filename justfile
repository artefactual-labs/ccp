set shell := ["bash", "-uc"]

[private]
default:
  @just --list --unsorted

e2e-dump:
  dagger call --progress=plain --source=".:default" generate-dumps export --path=hack/ccp/e2e/testdata/dumps

e2e:
  dagger call --progress=plain --source=".:default" etoe

amflow:
  amflow edit --file ./hack/ccp/internal/workflow/assets/workflow.json

grpcui:
  grpcui -plaintext -H "Authorization: ApiKey test:test" localhost:63030

run:
  make -C hack run

transfer:
  ./hack/ccp/hack/transfer-via-api.sh
