FROM golang:1.26 AS modules
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
FROM modules AS sources
COPY gen ./gen
COPY internal ./internal
COPY server/cmd ./server/cmd

# This file changes only when the imported dependency set changes.
RUN CGO_ENABLED=1 go list -deps -f '{{if and .Module (not .Standard)}}{{if not .Module.Main}}{{.ImportPath}}{{end}}{{end}}' \
    ./server/cmd/server ./server/cmd/collector ./server/cmd/aggregation-worker ./server/cmd/migrate-sqlite-postgres \
    > /tmp/eye-dependencies.unsorted && sort -u /tmp/eye-dependencies.unsorted | sed '/^$/d' > /tmp/eye-dependencies

FROM modules AS build
COPY --from=sources /tmp/eye-dependencies /tmp/eye-dependencies
RUN CGO_ENABLED=1 xargs -r go build < /tmp/eye-dependencies
COPY --from=sources /src /src
RUN CGO_ENABLED=1 go build -o /out/ \
    ./server/cmd/server ./server/cmd/collector ./server/cmd/aggregation-worker ./server/cmd/migrate-sqlite-postgres
RUN mv /out/server /out/eye-server && mv /out/collector /out/eye-collector && \
    mv /out/aggregation-worker /out/eye-aggregation-worker && mv /out/migrate-sqlite-postgres /out/eye-migrate

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget && rm -rf /var/lib/apt/lists/*
RUN useradd --system --uid 10001 eye
WORKDIR /app
COPY --from=build /out/eye-server /app/eye-server
COPY --from=build /out/eye-collector /app/eye-collector
COPY --from=build /out/eye-aggregation-worker /app/eye-aggregation-worker
COPY --from=build /out/eye-migrate /app/eye-migrate
RUN mkdir -p /data/snapshots && chown -R eye:eye /data
USER eye
ENV EYE_ENV=production DB_PATH=/data/eye.db SNAPSHOT_DIR=/data/snapshots PORT=8080
EXPOSE 8080 50051 50052
CMD ["/app/eye-server"]
