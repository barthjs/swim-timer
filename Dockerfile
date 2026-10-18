FROM --platform=$BUILDPLATFORM node:24-alpine AS web-builder
WORKDIR /build/web

ARG VERSION=dev
ENV VITE_APP_VERSION=${VERSION}

COPY web/package*.json ./
RUN npm ci --omit=dev

COPY web/ ./
RUN npm run build-only

FROM --platform=$BUILDPLATFORM node:24-alpine AS apidocs-builder
WORKDIR /build/docs/api

COPY docs/api/package*.json ./
RUN npm ci --omit=dev

COPY docs/api/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS api-builder
WORKDIR /build

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /build/web/dist ./web/dist
COPY --from=apidocs-builder /build/docs/api/dist ./docs/api/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X 'main.Version=${VERSION}'" -o ./bin/swim-timer ./cmd/swim-timer/

FROM alpine:latest AS runtime

ENV APP_USER=application \
    PUID=1000 \
    PGID=1000 \
    CONFIG_FILE=/data/config.yaml \
    LOG_PATH=/data/logs \
    STORAGE_PATH=/data/storage \
    XDG_CONFIG_HOME=/data \
    SWIM_TIMER_SUPERVISED=1

RUN set -eux \
    && addgroup -g ${PGID} ${APP_USER} \
    && adduser -D -u ${PUID} -G ${APP_USER} -h /home/${APP_USER} -s /bin/sh ${APP_USER} \
    && apk add --no-cache \
    shadow \
    su-exec

COPY scripts/entrypoint.sh /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]

CMD ["swim-timer"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["swim-timer", "healthcheck"]

# GoReleaser builds this target from the binaries of its own build, which it places under the target platform.
FROM runtime AS release

ARG TARGETPLATFORM

COPY ${TARGETPLATFORM}/swim-timer /usr/local/bin

FROM runtime AS prod

COPY --from=api-builder /build/bin/swim-timer /usr/local/bin
