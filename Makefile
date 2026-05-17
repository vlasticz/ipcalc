.PHONY: dev build test css css-watch vendor docker docker-amd64 docker-multiarch clean help

# ---- Configuration ----
BINARY      := server
PKG         := ./cmd/server
OUT         := bin/$(BINARY)
IMAGE       := ipcalc
TAG         := dev
PLATFORM_ARM:= linux/arm64
PLATFORM_AMD:= linux/amd64
HTMX_VERSION:= 2.0.4

# ---- Local dev ----
dev: ## Run with hot reload (Air for Go, Tailwind CLI in watch mode).
	@command -v air >/dev/null 2>&1 || { echo "air not installed; install with: go install github.com/air-verse/air@latest"; exit 1; }
	@command -v tailwindcss >/dev/null 2>&1 || { echo "tailwindcss CLI not installed; install with: npm i -g @tailwindcss/cli or download the standalone binary"; exit 1; }
	@trap 'kill 0' INT TERM EXIT; tailwindcss -i input.css -o static/app.css --watch & air

# ---- Build ----
build: css ## Build the static binary for the host platform.
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(OUT) $(PKG)

css: ## Build production Tailwind CSS (purged).
	@command -v tailwindcss >/dev/null 2>&1 || { echo "tailwindcss CLI not installed"; exit 1; }
	tailwindcss -i input.css -o static/app.css --minify

css-watch: ## Tailwind CSS in watch mode (without Air).
	tailwindcss -i input.css -o static/app.css --watch

# ---- Test ----
test: ## Run all Go tests.
	go test -race -count=1 ./...

# ---- Vendoring ----
vendor: ## Download / refresh vendored frontend assets (htmx).
	@mkdir -p static
	curl -fsSL "https://unpkg.com/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js" -o static/htmx.min.js
	@echo "Vendored htmx $(HTMX_VERSION) -> static/htmx.min.js"

# ---- Docker ----
docker: ## Build the ARM64 image (primary target).
	docker buildx build --platform $(PLATFORM_ARM) --load -t $(IMAGE):$(TAG) .

docker-amd64: ## Build the AMD64 image.
	docker buildx build --platform $(PLATFORM_AMD) --load -t $(IMAGE):$(TAG)-amd64 .

docker-multiarch: ## Build both architectures (no --load; requires a registry to push).
	docker buildx build --platform $(PLATFORM_ARM),$(PLATFORM_AMD) -t $(IMAGE):$(TAG) .

# ---- Housekeeping ----
clean:
	rm -rf bin/ dist/ static/app.css

help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
