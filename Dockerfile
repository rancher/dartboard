ARG K6_VERSION="0.58.0"
FROM golang:1.24-alpine3.20 AS builder
# match whichever tagged version is used by the K6_VERSION docker image
# see build layer at https://github.com/grafana/k6/blob/v${K6_VERSION}/Dockerfile

ENV WORKSPACE=/src/dartboard/
WORKDIR $WORKSPACE

COPY [".", "$WORKSPACE"]

RUN apk update && \
    apk add bash tar unzip wget curl make

RUN go mod download && \
    go mod tidy && \
    go mod verify

RUN cd $WORKSPACE && \
    make && \
    mv ./dartboard /usr/local/bin/dartboard

FROM grafana/k6:${K6_VERSION}
COPY --from=builder /usr/local/bin/dartboard /bin/dartboard

# Run the following commands as root user so that we can easily install some needed tools
USER root

RUN apk update && \
    apk add --no-cache \
    openssh-client \
    netcat-openbsd \
    bash

# switch back to the non-root user
USER k6

CMD [ "dartboard" ]
