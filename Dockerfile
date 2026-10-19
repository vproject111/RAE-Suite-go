FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY pkg ./pkg
COPY services ./services
COPY cmd ./cmd
RUN mkdir -p /out && \
    for mod in rae-memory rae-crl rae-mcp rae-suite rae-quality rae-lab rae-phoenix rae-hive rae-portal rae; do \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/$mod ./cmd/$mod; \
    done

FROM alpine:3.20
ARG SERVICE_BIN=rae
WORKDIR /app
COPY --from=builder /out/ /usr/local/bin/
RUN ln -sf /bin/sh /bin/bash && \
    printf '#!/bin/sh\nexit 0\n' > /usr/local/bin/alembic && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-memory\n' > /usr/local/bin/uvicorn && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-mcp\n' > /usr/local/bin/python && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-mcp\n' > /usr/local/bin/python3 && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-crl\n' > /usr/local/bin/celery && \
    chmod +x /usr/local/bin/alembic /usr/local/bin/uvicorn /usr/local/bin/python /usr/local/bin/python3 /usr/local/bin/celery && \
    ln -sf /usr/local/bin/${SERVICE_BIN} /usr/local/bin/rae-entrypoint
EXPOSE 8000 8005 8009 8010 8011 8012 8013 8080 8082 8503
ENTRYPOINT ["/usr/local/bin/rae-entrypoint"]
