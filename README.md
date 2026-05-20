# ipcalc

[![Docker Pulls](https://img.shields.io/docker/pulls/vlasticz/ipcalc?logo=docker&label=docker%20pulls)](https://hub.docker.com/r/vlasticz/ipcalc)
[![Docker Image Size](https://img.shields.io/docker/image-size/vlasticz/ipcalc/latest?logo=docker&label=image%20size)](https://hub.docker.com/r/vlasticz/ipcalc)
[![License](https://img.shields.io/github/license/vlasticz/ipcalc)](LICENSE)

A modern, lightweight web-based IPv4 subnet calculator. Go + HTMX + Tailwind, single
static binary, multi-arch Docker image (arm64 / amd64).

## Features

- Single-subnet calculator with `kjokjo/ipcalc` parity plus modern
  warning chips (RFC1918, link-local, multicast, loopback, public)
- Subnetting: equal-N split and VLSM (variable-length, host-count list)
- Live HTMX updates, locale-aware number formatting
- `/healthz` endpoint and self-probing `-healthcheck` flag for container
  liveness — no `curl` needed in the runtime image

## Run

### Dev — hot reload

```sh
make dev    # requires Air + Tailwind CLI; watches Go + CSS in parallel
```

### Local binary

```sh
make build
./bin/server    # defaults: -port 8080, -db data/ipcalc.db
```

### Docker — pull from Hub

```sh
docker run --rm -p 8080:8080 -v "$PWD/data:/data" vlasticz/ipcalc:latest
```

Or with Compose:

```sh
cp compose.example.yaml compose.yaml    # edit if needed
docker compose up -d                    # pulls vlasticz/ipcalc:latest from Hub
```

### Docker — build from source

```sh
make docker                             # arm64 by default; -amd64 / -multiarch variants in the Makefile
docker run --rm -p 8080:8080 -v "$PWD/data:/data" ipcalc:dev
```

Or via Compose with a forced rebuild:

```sh
docker compose up --build -d            # uses the compose file's `build: .` directive
```

Without `--build`, Compose pulls from Hub even though `build: .` is set in the file.

## Credits

Calculation logic is a Go port of
[`kjokjo/ipcalc`](https://github.com/kjokjo/ipcalc), originally by
Krischan Jodies. Display conventions (binary representation, classful
labels, wildcard mask, `/31` and `/32` handling) match that reference.
Logic is ported, not shelled out.

## Licence

GPLv3. See [`LICENSE`](LICENSE).
