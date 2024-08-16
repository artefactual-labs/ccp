[private]
default:
  @just --list --unsorted

# Deploy prod overlay.
prod:
  kubectl kustomize overlays/prod | kubectl apply -f -

# Trigger a rolling restart of the ccp deployment.
restart:
  kubectl rollout restart -n ccp deployment ccp

# Create a shell using the worker image.
shell-worker:
  kubectl run -it --rm --image=ghcr.io/artefactual-labs/ccp:v2.0.0-beta.4 --namespace ccp debug-shell --command -- /bin/bash

# Create a shell using the alpine image.
shell-alpine:
  kubectl run -it --rm --image=alpine:3.20.2 --namespace ccp debug-shell --command -- /bin/sh

shell-db:
  kubectl run -it --rm --image=mysql:8.0 --namespace ccp debug-shell --command -- mysql -hccp-mysql.ccp.svc.cluster.local -uroot -padmin

# Create the install job.
install:
  kubectl create job --from=cronjob/ccp-install ccp-install

# Remove all the resources within the ccp namespace.
flush-ccp-namespace:
  kubectl delete all --all --namespace ccp
