FROM golang:1.26.10-alpine AS builder
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
RUN addgroup -g 10001 -S raeuser && \
    adduser -u 10001 -S -G raeuser -H -h /app raeuser && \
    mkdir -p /app/data /tmp/rae-suite-store && \
    chown -R raeuser:raeuser /app /tmp/rae-suite-store && \
    ln -sf /bin/sh /bin/bash && \
    printf '#!/bin/sh\nexit 0\n' > /usr/local/bin/alembic && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-memory\n' > /usr/local/bin/uvicorn && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-mcp\n' > /usr/local/bin/python && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-mcp\n' > /usr/local/bin/python3 && \
    printf '#!/bin/sh\nexec /usr/local/bin/rae-crl\n' > /usr/local/bin/celery && \
    chmod +x /usr/local/bin/alembic /usr/local/bin/uvicorn /usr/local/bin/python /usr/local/bin/python3 /usr/local/bin/celery && \
    ln -sf /usr/local/bin/${SERVICE_BIN} /usr/local/bin/rae-entrypoint
USER raeuser
EXPOSE 8000 8005 8009 8010 8011 8012 8013 8080 8082 8503
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD wget -qO- http://127.0.0.1:${PORT:-8000}/health || exit 1
ENTRYPOINT ["/usr/local/bin/rae-entrypoint"]
