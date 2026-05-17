# ipcalc

A modern web-based IPv4 subnet calculator. Go + HTMX + Tailwind, single static
binary, ARM64/AMD64 Docker image.

## Status

Pre-MVP scaffold. See `cmd/server/main.go` for the entry point.

## Credits

The calculation logic is a Go port of [`kjokjo/ipcalc`](https://github.com/kjokjo/ipcalc),
originally authored by Krischan Jodies (the canonical Perl `ipcalc`). The
display conventions (binary representation, classful labels, wildcard mask,
`/31` and `/32` handling) match that reference implementation.

Logic is ported, not shelled out — the reference is read for correctness, not
invoked at runtime.

## Licence

GPLv3. See [`LICENSE`](LICENSE).
