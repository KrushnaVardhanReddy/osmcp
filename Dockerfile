FROM alpine:3.19

RUN apk add --no-cache git bash

LABEL io.modelcontextprotocol.server.name="io.github.KrushnaVardhanReddy/osmcp"

COPY osmcp /usr/local/bin/osmcp
ENTRYPOINT ["/usr/local/bin/osmcp"]
