ARG K6_VERSION="1.7.1"
FROM golang:1.24.2-alpine3.21 AS builder
# match whichever tagged version is used by the K6_VERSION docker image
# see build layer at https://github.com/grafana/k6/blob/v${K6_VERSION}/Dockerfile

ENV WORKSPACE=/dartboard/
WORKDIR $WORKSPACE

COPY [".", "$WORKSPACE"]

RUN apk add --no-cache \
    bash=5.3.3-r1 \
    tar=1.35-r4 \
    unzip=6.0-r16 \
    wget=1.25.0-r3 \
    curl=8.19.0-r0 \
    make=4.4.1-r4

RUN go mod download && \
    go mod tidy && \
    go mod verify

RUN cd $WORKSPACE && \
    make && \
    mv ./dartboard /usr/local/bin/dartboard && \
    mv ./qasereporter-k6/qasereporter-k6 /usr/local/bin/qasereporter-k6

FROM grafana/k6:${K6_VERSION}
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
