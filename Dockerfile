# syntax=docker/dockerfile:1.7-labs
FROM node:24-alpine AS node-builder
WORKDIR /build/web

ARG VERSION=dev
ENV VITE_APP_VERSION=${VERSION}

COPY web/package*.json ./
RUN npm ci --omit=dev

COPY web/ ./
RUN npm run build-only

FROM golang:1.26-alpine AS go-builder
WORKDIR /build

ARG VERSION=dev
ENV APP_VERSION=${VERSION}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=node-builder /build/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X 'main.Version=${APP_VERSION}'" -o ./swim-timer ./cmd/

FROM alpine:latest
WORKDIR /app

ENV APP_USER=application \
    PUID=1000 \
    PGID=1000 \
    PRESETS_DIR=/app/data/presets \
    RACE_DATA_DIR=/app/data/race-data

RUN set -eux \
    && addgroup -g ${PGID} ${APP_USER} \
    && adduser -D -u ${PUID} -G ${APP_USER} -h /home/${APP_USER} -s /bin/sh ${APP_USER} \
    && apk add --no-cache \
    shadow \
    su-exec

COPY --from=go-builder /build/swim-timer ./swim-timer

COPY entrypoint.sh /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]

CMD ["/app/swim-timer" , "--config", "/app/data/config.yaml"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -nv --spider http://localhost:8080/api/ping || exit 1
