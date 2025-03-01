FROM golang:1.24-alpine3.20@sha256:79f7ffeff943577c82791c9e125ab806f133ae3d8af8ad8916704922ddcc9dd8

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

# renovate: datasource=repology depName=alpine_3_20/ca-certificates
ENV CA_CERTIFICATES_VERSION="20241121-r1"
# renovate: datasource=repology depName=alpine_3_20/dpkg
ENV DPKG_VERSION="1.22.6-r1"
# renovate: datasource=repology depName=alpine_3_20/gnupg
ENV GNUPG_VERSION="2.4.5-r0"
# renovate: datasource=repology depName=alpine_3_20/curl
ENV CURL_VERSION="8.12.1-r0"
# renovate: datasource=repology depName=alpine_3_20/bash
ENV BASH_VERSION="5.2.26-r0"
# renovate: datasource=repology depName=alpine_3_20/openssl
ENV OPENSSL_VERSION="3.3.3-r0"

ENV GOSU_VERSION=1.17
RUN apk add --no-cache --virtual .gosu-deps \
    ca-certificates=="${CA_CERTIFICATES_VERSION}" \
    dpkg=="${DPKG_VERSION}" \
    gnupg=="${GNUPG_VERSION}" && \
	dpkgArch="$(dpkg --print-architecture | awk -F- '{ print $NF }')" && \
	wget -q -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch" && \
	wget -q -O /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch.asc" && \
	GNUPGHOME="$(mktemp -d)" && \
	export GNUPGHOME && \
	gpg --batch --keyserver hkps://keys.openpgp.org --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4 && \
	gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu && \
	gpgconf --kill all && \
	rm -rf "$GNUPGHOME" /usr/local/bin/gosu.asc && \
	apk del --no-network .gosu-deps && \
	chmod +x /usr/local/bin/gosu

WORKDIR /app

RUN adduser -S -G nobody riq

RUN apk update && \
    apk add --no-cache \
      curl=="${CURL_VERSION}" \
      bash=="${BASH_VERSION}" \
      ca-certificates=="${CA_CERTIFICATES_VERSION}" \
      openssl=="${OPENSSL_VERSION}"

COPY . ./

RUN chown -R riq:nobody /app

RUN go mod download && \
  go build -o /app/riq && \
  go clean -cache -modcache

ENTRYPOINT ["/bin/bash"]
CMD ["/app/docker-entrypoint.sh"]
