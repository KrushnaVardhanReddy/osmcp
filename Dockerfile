FROM alpine:3.19

RUN apk add --no-cache git bash

COPY osmcp /usr/local/bin/osmcp
ENTRYPOINT ["/usr/local/bin/osmcp"]
