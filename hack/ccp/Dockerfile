# syntax = docker/dockerfile:1.4

FROM node:20 AS node-builder
WORKDIR /app
COPY --link web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/app/.npm npm set cache /app/.npm && npm install-clean
COPY --link web/ ./
RUN npm run build

FROM golang:1.22.5-alpine AS go-builder
ARG VERSION_PATH=github.com/artefactual/archivematica/hack/ccp/internal/version
ARG VERSION_NUMBER
ARG VERSION_GIT_COMMIT
WORKDIR /src
ENV CGO_ENABLED=0
COPY --link go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link . .
COPY --from=node-builder /internal/webui/assets /src/internal/webui/assets
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
	go build \
	-ldflags="-X '${VERSION_PATH}.version=${VERSION_NUMBER}' -X '${VERSION_PATH}.gitCommit=${VERSION_GIT_COMMIT}'" \
	-trimpath \
	-o /out/ccp \
	.

FROM alpine:3.20.0 AS ccp
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S ccp
RUN adduser -u ${USER_ID} -S -D ccp ccp
COPY --from=go-builder --link /out/ccp /home/ccp/bin/ccp
USER ccp
CMD ["/home/ccp/bin/ccp", "server"]
