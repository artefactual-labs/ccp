ARG TARGET
ARG UBUNTU_VERSION=22.04
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG PYTHON_VERSION=3.12.4
ARG GO_VERSION=1.22.5
ARG NODE_VERSION=20
ARG PYENV_DIR=/pyenv
ARG SELENIUM_DIR=/selenium
ARG MEDIAAREA_VERSION=1.0-24

# -----------------------------------------------------------------------------

FROM ubuntu:${UBUNTU_VERSION} AS base-builder

ARG PYENV_DIR

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

RUN set -ex \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
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

ENV PYENV_ROOT=${PYENV_DIR}/data
ENV PATH=$PYENV_ROOT/shims:$PYENV_ROOT/bin:$PATH

# -----------------------------------------------------------------------------

FROM base-builder AS pyenv-builder

ARG PYTHON_VERSION

RUN set -ex \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends \
		build-essential \
		libbz2-dev \
		libffi-dev \
		liblzma-dev \
		libncursesw5-dev \
		libreadline-dev \
		libsqlite3-dev \
		libssl-dev \
		libxml2-dev \
		libxmlsec1-dev \
		tk-dev \
		xz-utils \
		zlib1g-dev \
	&& rm -rf /var/lib/apt/lists/* /var/cache/apt/*

RUN set -ex \
	&& curl --retry 3 -L https://github.com/pyenv/pyenv-installer/raw/master/bin/pyenv-installer | bash \
	&& pyenv install ${PYTHON_VERSION} \
	&& pyenv global ${PYTHON_VERSION}

COPY --link requirements-dev.txt /src/requirements-dev.txt

RUN set -ex \
	&& pyenv exec python3 -m pip install --upgrade pip setuptools \
	&& pyenv exec python3 -m pip install --requirement /src/requirements-dev.txt \
	&& pyenv rehash

# -----------------------------------------------------------------------------

FROM base-builder AS base

ARG USER_ID
ARG GROUP_ID
ARG PYENV_DIR
ARG MEDIAAREA_VERSION

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

RUN set -ex \
	&& groupadd --gid ${GROUP_ID} --system archivematica \
	&& useradd --uid ${USER_ID} --gid ${GROUP_ID} --home-dir /var/archivematica --system archivematica \
	&& mkdir -p /var/archivematica/sharedDirectory \
	&& chown -R archivematica:archivematica /var/archivematica

# Download ClamAV virus signatures
RUN freshclam --quiet

USER archivematica

COPY --chown=${USER_ID}:${GROUP_ID} --from=pyenv-builder --link ${PYENV_DIR} ${PYENV_DIR}
COPY --chown=${USER_ID}:${GROUP_ID} --link . /src

# -----------------------------------------------------------------------------

FROM base AS archivematica-mcp-client

ENV DJANGO_SETTINGS_MODULE=settings.common
ENV PYTHONPATH=/src/src/MCPClient/lib/:/src/src/MCPClient/lib/clientScripts

# Some scripts in archivematica-fpr-admin executed by MCPClient rely on certain
# files being available in this image (e.g. see https://git.io/vA1wF).
COPY --link src/MCPClient/externals/fido/ /usr/lib/archivematica/archivematicaCommon/externals/fido/
COPY --link src/MCPClient/externals/fiwalk_plugins/ /usr/lib/archivematica/archivematicaCommon/externals/fiwalk_plugins/

ENTRYPOINT ["pyenv", "exec", "python3", "/src/src/MCPClient/lib/archivematicaClient.py"]

# -----------------------------------------------------------------------------

FROM base AS archivematica-tests

# -----------------------------------------------------------------------------

FROM node:$NODE_VERSION AS node-builder
WORKDIR /app
COPY --link hack/ccp/web/package.json hack/ccp/web/package-lock.json ./
RUN --mount=type=cache,target=/app/.npm npm set cache /app/.npm && npm install-clean
COPY --link hack/ccp/web/ ./
RUN npm run build

# -----------------------------------------------------------------------------

FROM golang:$GO_VERSION-alpine AS go-builder
ARG VERSION_PATH=github.com/artefactual/archivematica/hack/ccp/internal/version
ARG VERSION_NUMBER
ARG VERSION_GIT_COMMIT
WORKDIR /src
ENV CGO_ENABLED=0
COPY --link hack/ccp/go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link hack/ccp/ ./
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

FROM ${TARGET}
