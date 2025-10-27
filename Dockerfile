ARG K6_VERSION="1.3.0"
FROM golang:1.24-alpine3.22 AS builder
# match whichever tagged version is used by the K6_VERSION docker image
# see build layer at https://github.com/grafana/k6/blob/v${K6_VERSION}/Dockerfile

ENV WORKSPACE=/dartboard/
WORKDIR $WORKSPACE

COPY [".", "$WORKSPACE"]

RUN apk update && \
    apk add bash tar unzip wget curl make

RUN go mod download && \
    go mod tidy && \
    go mod verify

RUN cd $WORKSPACE && \
    make && \
    mv ./dartboard /usr/local/bin/dartboard && \
    mv ./qasereporter-k6/qasereporter-k6 /usr/local/bin/qasereporter-k6

# Clean up unnecessary files to reduce image size
RUN rm -rf \
    /dartboard/docs \
    /dartboard/k6 \
    /dartboard/tofu \
    /dartboard/charts \
    /dartboard/scripts \
    /dartboard/darts \
    /dartboard/*.md

# Clean up all "hidden" files
RUN find . -maxdepth 1 -type f -name ".*" -delete

FROM grafana/k6:${K6_VERSION}
COPY --from=builder /usr/local/bin/dartboard /bin/dartboard
COPY --from=builder /usr/local/bin/qasereporter-k6 /bin/qasereporter-k6

# Run the following commands as root user so that we can easily install some needed tools
USER root

RUN apk update && \
    apk add --no-cache \
    openssh-client \
    netcat-openbsd \
    bash \
    gettext \
    zip unzip \
    yq

# switch back to the non-root user
USER k6

ENTRYPOINT [ "dartboard" ]
