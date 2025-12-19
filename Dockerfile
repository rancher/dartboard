# renovate: datasource=docker depName=grafana/k6
ARG K6_VERSION="1.7.1"
# renovate: datasource=docker depName=grafana/k6 digestVersion=1.7.1
ARG K6_IMAGE_DIGEST="sha256:4fd3a694926b064d3491d9b02b01cde886583c4931f1223816e3d9a7bdfa7e0f"
# renovate: datasource=docker depName=golang
ARG GOLANG_VERSION="1.24.2-alpine3.21"
# renovate: datasource=docker depName=golang digestVersion=1.24.2-alpine3.21
ARG GOLANG_IMAGE_DIGEST="sha256:7772cb5322baa875edd74705556d08f0eeca7b9c4b5367754ce3f2f00041ccee"
FROM golang:${GOLANG_VERSION}@${GOLANG_IMAGE_DIGEST} AS builder
# match whichever tagged version is used by the K6_VERSION docker image
# see build layer at https://github.com/grafana/k6/blob/v${K6_VERSION}/Dockerfile

ENV WORKSPACE=/dartboard/
WORKDIR $WORKSPACE

COPY [".", "$WORKSPACE"]

RUN apk update && \
    apk add --no-cache \
    bash~=5.2.37 \
    tar~=1.35 \
    unzip~=6.0 \
    wget~=1.25.0 \
    curl~=8.14.1 \
    make~=4.4.1

RUN go mod download && \
    go mod tidy && \
    go mod verify

RUN cd $WORKSPACE && \
    make && \
    mv ./dartboard /usr/local/bin/dartboard && \
    mv ./qasereporter-k6/qase-k6-cli /usr/local/bin/qase-k6-cli

FROM grafana/k6:${K6_VERSION}@${K6_IMAGE_DIGEST}
COPY --from=builder /usr/local/bin/dartboard /bin/dartboard
COPY --from=builder /usr/local/bin/qase-k6-cli /bin/qase-k6-cli

# Run the following commands as root user so that we can easily install some needed tools
USER root

RUN apk update && \
    apk add --no-cache \
    openssh-client~=10.2 \
    netcat-openbsd~=1.234.1 \
    bash~=5.3.3 \
    gettext~=0.24.1 \
    zip~=3.0 \
    unzip~=6.0 \
    yq-go~=4.49.2

# switch back to the non-root user
USER k6

ENTRYPOINT [ "dartboard" ]
