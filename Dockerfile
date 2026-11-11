FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/eye-server ./server/cmd/server
RUN CGO_ENABLED=1 go build -o /out/eye-collector ./server/cmd/collector

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
RUN useradd --system --uid 10001 eye
WORKDIR /app
COPY --from=build /out/eye-server /app/eye-server
COPY --from=build /out/eye-collector /app/eye-collector
RUN mkdir -p /data/snapshots && chown -R eye:eye /data
USER eye
ENV EYE_ENV=production DB_PATH=/data/eye.db SNAPSHOT_DIR=/data/snapshots PORT=8080
EXPOSE 8080 50051 50052
CMD ["/app/eye-server"]
