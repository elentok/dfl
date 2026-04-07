FROM alpine:3.20

RUN apk add --no-cache tar

WORKDIR /repo

COPY bootstrap ./bootstrap

RUN mkdir -p /fixture/bin /fixture/release \
  && printf '#!/bin/sh\ncase "$*" in\n  "-qO- "*dfl_Linux_x86_64.tar.gz) cat /fixture/release/dfl_Linux_x86_64.tar.gz ;;\n  "-qO- "*dfl_Linux_arm64.tar.gz) cat /fixture/release/dfl_Linux_arm64.tar.gz ;;\n  *) echo "unexpected wget args: $*" >&2; exit 1 ;;\nesac\n' > /fixture/bin/wget \
  && chmod +x /fixture/bin/wget \
  && printf '#!/bin/sh\n[ "$*" = "setup" ] || { echo "unexpected dfl args: $*" >&2; exit 1; }\necho "fake dfl setup ran"\n' > /fixture/release/dfl \
  && tar -czf /fixture/release/dfl_Linux_x86_64.tar.gz -C /fixture/release dfl \
  && tar -czf /fixture/release/dfl_Linux_arm64.tar.gz -C /fixture/release dfl \
  && rm /fixture/release/dfl

RUN PATH="/fixture/bin:/bin:/usr/bin" HOME=/tmp/dfl-home ./bootstrap
