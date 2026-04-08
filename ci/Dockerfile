# ─── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder
ARG APP_NAME=go-template
LABEL stage=builder-${APP_NAME}

WORKDIR /go/src/github.com/ganasa18/${APP_NAME}

# Download deps first (layer-cached unless go.mod/go.sum change).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Statically linked binary — no libc dependency in the final image.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -ldflags='-w -s -extldflags "-static"' -o /${APP_NAME} .

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM alpine:latest
ARG APP_NAME=go-template

ENV SERVICE_NAME=${APP_NAME}
ENV APP_HOME=/opt/app/${APP_NAME}
ENV TZ=Asia/Jakarta

LABEL stage=runtime-${APP_NAME}

WORKDIR ${APP_HOME}

# Copy binary and startup script.
COPY --from=builder /${APP_NAME} .
COPY ci/startup.sh /opt/startup.sh
# Copy env template (populated at runtime by startup.sh from /opt/secret-volume).
COPY ci/env.conf ${APP_HOME}/temp/env.conf

RUN addgroup --gid 2001 --system appgroup \
    && adduser --uid 2001 --system --ingroup appgroup appuser \
    && apk --no-cache update \
    && apk --no-cache add ca-certificates bash jq curl tzdata \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone \
    && chmod +x ${APP_HOME}/${APP_NAME} /opt/startup.sh \
    && mkdir -p /opt/secret-volume \
    && chown appuser:appgroup ${APP_HOME} /opt/secret-volume -R

USER appuser

EXPOSE 8899

ENTRYPOINT ["/opt/startup.sh"]
