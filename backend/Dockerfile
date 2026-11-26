# ---- builder ----
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
RUN go install mvdan.cc/garble@v0.15.0
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN GOGARBLE=github.com/renankeny/conversion-tracker/backend CGO_ENABLED=0 GOOS=linux garble -literals -seed=random build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN GOGARBLE=github.com/renankeny/conversion-tracker/backend CGO_ENABLED=0 GOOS=linux garble -literals -seed=random build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---- api ----
FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime && \
    echo "America/Sao_Paulo" > /etc/timezone && \
    addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /app/data && chown -R appuser:appgroup /app
ENV TZ=America/Sao_Paulo
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /out/api /app/api
COPY --chown=appuser:appgroup migrations /app/migrations
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/api"]

# ---- worker ----
FROM alpine:3.20 AS worker
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime && \
    echo "America/Sao_Paulo" > /etc/timezone && \
    addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /app/data && chown -R appuser:appgroup /app
ENV TZ=America/Sao_Paulo
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /out/worker /app/worker
USER appuser
ENTRYPOINT ["/app/worker"]
