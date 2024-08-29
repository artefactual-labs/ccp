ARG UBUNTU_VERSION=22.04
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG PYTHON_VERSION=3.12.5
ARG GO_VERSION=1.22.5
ARG UV_VERSION=0.4.0
ARG NODE_VERSION=20
ARG MEDIAAREA_VERSION=1.0-24

# -----------------------------------------------------------------------------

FROM ubuntu:${UBUNTU_VERSION} AS archivematica-worker-base

ARG USER_ID
ARG GROUP_ID
ARG MEDIAAREA_VERSION

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

RUN set -ex \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
		clang \
		curl \
		git \
		gnupg \
		libldap2-dev \
		libmysqlclient-dev \
		libsasl2-dev \
		libsqlite3-dev \
		locales \
		make \
		pkg-config \
		tzdata \
	&& rm -rf /var/lib/apt/lists/* /var/cache/apt/*

RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

RUN set -ex \
	&& curl --retry 3 -fsSL https://packages.archivematica.org/1.16.x/key.asc | gpg --dearmor -o /etc/apt/keyrings/archivematica-1.16.x.gpg \
	&& echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/archivematica-1.16.x.gpg] http://packages.archivematica.org/1.16.x/ubuntu-externals jammy main" > /etc/apt/sources.list.d/archivematica-external.list \
	&& curl --retry 3 -so /tmp/repo-mediaarea.deb -L https://mediaarea.net/repo/deb/repo-mediaarea_${MEDIAAREA_VERSION}_all.deb \
	&& dpkg -i /tmp/repo-mediaarea.deb \
	&& rm /tmp/repo-mediaarea.deb \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends \
		atool \
		bulk-extractor \
		clamav \
		coreutils \
		ffmpeg \
		fits \
		g++ \
		gcc \
		gearman \
		gettext \
		ghostscript \
		hashdeep \
		imagemagick \
		inkscape \
		jhove \
		libffi-dev \
		libimage-exiftool-perl \
		libldap2-dev \
		libmysqlclient-dev \
		libsasl2-dev \
		libssl-dev \
		libxml2-dev \
		libxslt1-dev \
		logapp \
		md5deep \
		mediaconch \
		mediainfo \
		nailgun \
		nfs-common \
		openjdk-8-jre-headless \
		p7zip-full \
		pbzip2 \
		pst-utils \
		python3-lxml \
		rsync \
		siegfried \
		sleuthkit \
		tesseract-ocr \
		tree \
		unar \
		unrar-free \
		uuid \
	&& rm -rf /var/lib/apt/lists/* /var/cache/apt/*

# Download ClamAV virus signatures.
RUN freshclam --quiet

# -----------------------------------------------------------------------------

FROM ghcr.io/astral-sh/uv:${UV_VERSION} AS uv

# -----------------------------------------------------------------------------

FROM archivematica-worker-base AS archivematica-worker

ARG PYTHON_VERSION

# Install Python.
COPY --from=uv /uv /bin/uv
ENV UV_PYTHON=${PYTHON_VERSION}
ENV UV_PYTHON_PREFERENCE=only-managed
RUN --mount=type=cache,target=/root/.cache/uv uv python install

# Use a regular user.
RUN set -ex \
	&& groupadd --gid ${GROUP_ID} --system archivematica \
	&& useradd --uid ${USER_ID} --gid ${GROUP_ID} --home-dir /var/archivematica --system archivematica \
	&& mkdir -p /var/archivematica/sharedDirectory \
	&& chown -R archivematica:archivematica /var/archivematica
USER archivematica

# Create virtual environment.
RUN --mount=type=cache,target=/root/.cache/uv uv venv /var/archivematica/venv
ENV PATH="/var/archivematica/venv/bin:$PATH"

WORKDIR /src

# Install requirements.
ADD worker/requirements/requirements-dev.txt requirements/requirements-dev.txt
RUN --mount=type=cache,target=/root/.cache/uv uv pip install -r requirements/requirements-dev.txt

# Copy the sources.
COPY --chown=${USER_ID}:${GROUP_ID} --link ./worker /src

# Assets needed by FPR scripts.
COPY --link worker/externals/fido/ /usr/lib/archivematica/archivematicaCommon/externals/fido/
COPY --link worker/externals/fiwalk_plugins/ /usr/lib/archivematica/archivematicaCommon/externals/fiwalk_plugins/

ENTRYPOINT ["python", "-m", "worker"]

# -----------------------------------------------------------------------------

FROM base AS archivematica-tests

# -----------------------------------------------------------------------------

FROM node:$NODE_VERSION AS node-builder
WORKDIR /app
COPY --link web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/app/.npm npm set cache /app/.npm && npm install-clean
COPY --link web/ ./
RUN npm run build

# -----------------------------------------------------------------------------

FROM golang:$GO_VERSION-alpine AS go-builder
ARG VERSION_PATH=github.com/artefactual-labs/ccp/internal/version
ARG VERSION_NUMBER
ARG VERSION_GIT_COMMIT
WORKDIR /src
ENV CGO_ENABLED=0
COPY --link go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link / ./
COPY --from=node-builder /internal/webui/assets /src/internal/webui/assets
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
	go build \
	-ldflags="-X '${VERSION_PATH}.version=${VERSION_NUMBER}' -X '${VERSION_PATH}.gitCommit=${VERSION_GIT_COMMIT}'" \
	-trimpath \
	-o /out/ccp \
	.

# -----------------------------------------------------------------------------

FROM alpine:3.20.0 AS archivematica-ccp
ARG USER_ID
ARG GROUP_ID
RUN addgroup -g ${GROUP_ID} -S ccp
RUN adduser -u ${USER_ID} -S -D ccp ccp
COPY --from=go-builder --link /out/ccp /home/ccp/bin/ccp
USER ccp
CMD ["/home/ccp/bin/ccp", "server"]

# -----------------------------------------------------------------------------

FROM archivematica-worker AS archivematica-full
COPY --from=go-builder --link /out/ccp /var/archivematica/bin/ccp
CMD ["/var/archivematica/bin/ccp", "server"]
