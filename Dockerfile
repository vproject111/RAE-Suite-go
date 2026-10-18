FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY pkg ./pkg
COPY services ./services
COPY cmd ./cmd
RUN mkdir -p /out && \
    for mod in rae-memory rae-crl rae-mcp rae-suite rae-quality rae-lab rae-phoenix rae-hive rae-portal rae; do \
        CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/$mod ./cmd/$mod; \
    done

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/ /usr/local/bin/
EXPOSE 8000 8005 8009 8010 8011 8012 8013 8082 8503
ENTRYPOINT ["/usr/local/bin/rae"]
CMD ["--all", "--profile=edge"]
