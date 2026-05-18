# ipcalc

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

**Dev** — hot reload via Air + Tailwind watch:

```sh
make dev
```

**Local binary**:

```sh
make build
./bin/server -port 8080 -db data/ipcalc.db
```

**Docker (arm64 by default; `make docker-amd64` or `docker-multiarch` for
others)**:

```sh
make docker
docker run --rm -p 8080:8080 -v "$PWD/data:/data" ipcalc:dev
```

The SQLite database lives at `/data/ipcalc.db` inside the container.
Bind-mount any host directory there and you're set — no UID juggling.

**Compose** — copy `compose.example.yaml` to `compose.yaml`, adjust ports
and volumes for your environment, then `docker compose up -d`.

## Credits

Calculation logic is a Go port of
[`kjokjo/ipcalc`](https://github.com/kjokjo/ipcalc), originally by
Krischan Jodies. Display conventions (binary representation, classful
labels, wildcard mask, `/31` and `/32` handling) match that reference.
Logic is ported, not shelled out.

## Licence

GPLv3. See [`LICENSE`](LICENSE).
