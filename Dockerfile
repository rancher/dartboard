# renovate: datasource=docker depName=grafana/k6
ARG K6_VERSION="2.3.0"
# renovate: datasource=docker depName=grafana/k6 digestVersion=2.3.0
ARG K6_IMAGE_DIGEST="sha256:9c2dee7f8ed74d317e4027c06a10f169b625638189de8d4555d0b3486a5aeb34"
# renovate: datasource=docker depName=golang
ARG GOLANG_VERSION="1.27.1-alpine3.24"
# renovate: datasource=docker depName=golang digestVersion=1.27.1-alpine3.24
ARG GOLANG_IMAGE_DIGEST="sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414"
FROM golang:${GOLANG_VERSION}@${GOLANG_IMAGE_DIGEST} AS builder
# match whichever tagged version is used by the K6_VERSION docker image
# see build layer at https://github.com/grafana/k6/blob/v${K6_VERSION}/Dockerfile

ENV WORKSPACE=/dartboard/
WORKDIR $WORKSPACE

RUN apk add --no-cache \
    # renovate: datasource=repology depName=alpine_3_24/bash versioning=loose
    bash~=5.3.9-r1 \
    # renovate: datasource=repology depName=alpine_3_24/tar versioning=loose
    tar~=1.35-r5 \
    # renovate: datasource=repology depName=alpine_3_24/unzip versioning=loose
    unzip~=6.0-r16 \
    # renovate: datasource=repology depName=alpine_3_24/wget versioning=loose
    wget~=1.25.0-r3 \
    # renovate: datasource=repology depName=alpine_3_24/curl versioning=loose
    curl~=8.22.0-r0 \
    # renovate: datasource=repology depName=alpine_3_24/make versioning=loose
    make~=4.4.1-r4

COPY [".", "$WORKSPACE"]

RUN go mod download && \
    go mod tidy && \
    go mod verify

RUN cd $WORKSPACE && \
    make && \
    mv ./dartboard /usr/local/bin/dartboard && \
    mv ./qase-k6-cli/qase-k6-cli /usr/local/bin/qase-k6-cli && \
    go clean -cache -modcache && \
    rm -rf /root/.cache /go/pkg/* /dartboard/*

FROM grafana/k6:${K6_VERSION}@${K6_IMAGE_DIGEST}
COPY --from=builder /usr/local/bin/dartboard /bin/dartboard
COPY --from=builder /usr/local/bin/qase-k6-cli /bin/qase-k6-cli

# Run the following commands as root user so that we can easily install some needed tools
USER root

RUN apk update && \
    apk add --no-cache \
    # renovate: datasource=repology depName=alpine_3_24/openssh-client versioning=loose
    openssh-client~=10.3_p1-r1 \
    # renovate: datasource=repology depName=alpine_3_24/netcat-openbsd versioning=loose
    netcat-openbsd~=1.234.1-r0 \
    # renovate: datasource=repology depName=alpine_3_24/bash versioning=loose
    bash~=5.3.9-r1 \
    # renovate: datasource=repology depName=alpine_3_24/gettext versioning=loose
    gettext~=1.0-r0 \
    # renovate: datasource=repology depName=alpine_3_24/zip versioning=loose
    zip~=3.0-r13 \
    # renovate: datasource=repology depName=alpine_3_24/unzip versioning=loose
    unzip~=6.0-r16 \
    # renovate: datasource=repology depName=alpine_3_24/yq-go versioning=loose
    yq-go~=4.53.3-r1

# switch back to the non-root user
USER k6

ENTRYPOINT [ "dartboard" ]
