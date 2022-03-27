FROM alpine:3.15.0 AS furiko-base

RUN addgroup -S furiko-io && adduser -S furiko -G furiko-io
WORKDIR /home/furiko

# Install various tools in the base image.
# NOTE(irvinlim): This installs the latest tz database at time of build.
RUN apk add --update --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

FROM furiko-base AS execution-controller
USER furiko
COPY execution-controller /
ENTRYPOINT [ "/execution-controller" ]

FROM furiko-base AS execution-webhook
USER furiko
COPY execution-webhook /
ENTRYPOINT [ "/execution-webhook" ]
