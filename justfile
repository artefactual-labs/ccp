[private]
default:
  @just --list --unsorted

# Deploy prod overlay.
prod:
  kubectl kustomize overlays/prod | kubectl apply -f -

# Trigger a rolling restart of the ccp deployment.
restart:
  kubectl rollout restart -n ccp deployment ccp

# Run a shell.
debug-shell:
  kubectl run -it --rm --image=ghcr.io/artefactual-labs/ccp:v2.0.0-beta.4 --namespace ccp debug-shell --command -- /bin/bash

migrate:
  kubectl create job --from=cronjob/ccp-migrate ccp-migrate

install:
  kubectl create job --from=cronjob/ccp-install ccp-install
