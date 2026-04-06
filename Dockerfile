# 1.7.1 linux/amd64
ARG K6_IMAGE_DIGEST="sha256:4fd3a694926b064d3491d9b02b01cde886583c4931f1223816e3d9a7bdfa7e0f"
# golang:1.24.2-alpine3.21 linux/amd64
FROM golang@sha256:7772cb5322baa875edd74705556d08f0eeca7b9c4b5367754ce3f2f00041ccee AS builder
# match whichever tagged version is used by the K6_VERSION docker image
# see build layer at https://github.com/grafana/k6/blob/v${K6_VERSION}/Dockerfile

ENV WORKSPACE=/dartboard/
WORKDIR $WORKSPACE

COPY [".", "$WORKSPACE"]

RUN apk add --no-cache \
    bash=5.2.37-r0 \
    tar=1.35-r2 \
    unzip=6.0-r15 \
    wget=1.25.0-r0 \
    curl=8.14.1-r2 \
    make=4.4.1-r2

RUN go mod download && \
    go mod tidy && \
    go mod verify

RUN cd $WORKSPACE && \
    make && \
    mv ./dartboard /usr/local/bin/dartboard && \
    mv ./qasereporter-k6/qasereporter-k6 /usr/local/bin/qasereporter-k6

FROM grafana/k6@${K6_IMAGE_DIGEST}
COPY --from=builder /usr/local/bin/dartboard /bin/dartboard
COPY --from=builder /usr/local/bin/qasereporter-k6 /bin/qasereporter-k6

# Run the following commands as root user so that we can easily install some needed tools
USER root

RUN apk update && \
    apk add --no-cache \
    openssh-client=10.2_p1-r0 \
    netcat-openbsd=1.234.1-r0 \
    bash=5.3.3-r1 \
    gettext=0.24.1-r1 \
    zip=3.0-r13 \
    unzip=6.0-r16 \
    yq-go=4.49.2-r4

# switch back to the non-root user
USER k6

ENTRYPOINT [ "dartboard" ]
