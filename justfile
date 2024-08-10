set shell := ["bash", "-uc"]

default:
  @just --list --unsorted

e2e-dump:
  dagger call --source=".:default" generate-dumps export --path=hack/ccp/e2e/testdata/dumps

e2e:
  dagger call --source=".:default" etoe

amflow:
  amflow edit --file ./hack/ccp/internal/workflow/assets/workflow.json

grpcui:
  grpcui -plaintext -H "Authorization: ApiKey test:test" localhost:63030
