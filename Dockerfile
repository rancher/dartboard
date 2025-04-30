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

# Prepare the k6 user's .ssh directory with correct ownership & perms
RUN mkdir -p /home/k6/.ssh \
 && chmod 700 /home/k6/.ssh \
 && chown k6:k6 /home/k6/.ssh

# switch back to the non-root user
USER k6

ENTRYPOINT [ "dartboard" ]
