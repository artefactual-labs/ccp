## Introduction

This is a development environment based on Docker Compose. It runs MySQL, CCP
and MCPClient.

## Getting started

This should be enough:

```
# Downloads the sampledata submodule for testing.
git submodule update --init --recursive

# Mount ~/.ccp/data inside the ccp container.
make create-volumes

# Build the Compose services.
make build

# Lunch CCP in the foreground.
make run
```

## Submit a transfer

Using the API:

    ./hack/helpers/transfer-via-api.sh

Using watched directories (to be removed):

    ./hack/helpers/transfer-via-watched-dir.sh

## Local dependencies

The basics:

- go
- just
- make
- dagger
- uv
