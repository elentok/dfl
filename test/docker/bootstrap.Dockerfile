FROM alpine:3.20

RUN apk add --no-cache tar wget

WORKDIR /repo

COPY bootstrap ./bootstrap

RUN mkdir -p .git \
  && printf '#!/bin/sh\nexit 0\n' > setup \
  && chmod +x setup

RUN HOME=/tmp/dfl-home DFL_ROOT=/repo PATH="/bin:/usr/bin" ./bootstrap
