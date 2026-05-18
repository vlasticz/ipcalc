# syntax=docker/dockerfile:1.7
#
# Multi-stage, multi-arch (linux/arm64 + linux/amd64) build.
# Build stage runs natively on $BUILDPLATFORM; Go cross-compiles for $TARGETPLATFORM.
# Final image is distroless/static — no shell, no package manager, ~15-20 MB.
#
# Build:        docker buildx build --platform linux/arm64 --load -t ipcalc:dev .
# Both archs:   docker buildx build --platform linux/arm64,linux/amd64 -t ipcalc:dev .

ARG GO_VERSION=1.24
ARG TAILWIND_VERSION=4.0.0
# Distroless static without the `:nonroot` suffix runs as root — required
# for bind-mounted /data to be writable out-of-the-box regardless of
# host UID. The image still has no shell, package manager, or libc; the
# attack surface is essentially "the single static Go binary".
ARG DISTROLESS_TAG=latest

# ----------------------------------------------------------------------------
# Stage 1: build CSS + Go binary
# ----------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build

ARG TAILWIND_VERSION
ARG BUILDARCH
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Tailwind CLI standalone binary (no Node runtime). Downloaded for the BUILD
# arch since it runs here, not in the final image.
RUN set -eux; \
    case "${BUILDARCH}" in \
      amd64) tw_arch="x64" ;; \
      arm64) tw_arch="arm64" ;; \
      *) echo "unsupported BUILDARCH: ${BUILDARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /usr/local/bin/tailwindcss \
      "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-linux-${tw_arch}"; \
    chmod +x /usr/local/bin/tailwindcss; \
    tailwindcss --help >/dev/null

# Go module download cached separately from source.
COPY go.mod go.sum* ./
RUN go mod download

# Source.
COPY . .

# Build CSS first so the binary can embed the static dir if we choose to.
RUN tailwindcss -i input.css -o static/app.css --minify

# Cross-compile the Go binary.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -trimpath -o /out/server ./cmd/server

# ----------------------------------------------------------------------------
# Stage 2: minimal runtime
# ----------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:${DISTROLESS_TAG}

ENV PORT=8080 \
    DB_PATH=/data/ipcalc.db

COPY --from=build /out/server /server
COPY --from=build /src/static /static

EXPOSE 8080
VOLUME ["/data"]

# Single-binary healthcheck — distroless/static has no shell or curl, so the
# server itself implements the probe behind a flag.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/server", "-healthcheck"]

ENTRYPOINT ["/server"]
