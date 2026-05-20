.PHONY: dev build test css css-watch vendor docker docker-amd64 docker-multiarch scan publish clean help

# ---- Configuration ----
BINARY      := server
PKG         := ./cmd/server
OUT         := bin/$(BINARY)
IMAGE       := ipcalc
TAG         := dev
HUB_IMAGE   := vlasticz/ipcalc
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

# ---- Release pipeline ----
scan: ## Build a local amd64 image (--load) and run Docker Scout's CVE quickview. Pre-release sanity check.
	@command -v docker >/dev/null 2>&1 || { echo "docker not installed"; exit 1; }
	docker buildx build --platform $(PLATFORM_AMD) --pull --load -t $(IMAGE):scan .
	@echo ""
	@echo "=== Docker Scout — quickview ==="
	docker scout quickview $(IMAGE):scan
	@echo ""
	@echo "Drill down with:  docker scout cves $(IMAGE):scan"

publish: ## Build multi-arch and push to Docker Hub. Required: VERSION=x.y.z. Also tags :latest.
	@if [ -z "$(VERSION)" ]; then \
	  echo "VERSION required, e.g.: make publish VERSION=0.9.1"; exit 1; \
	fi
	docker buildx build \
	  --platform $(PLATFORM_ARM),$(PLATFORM_AMD) \
	  --pull \
	  -t $(HUB_IMAGE):$(VERSION) \
	  -t $(HUB_IMAGE):latest \
	  --push .
	@echo ""
	@echo "Pushed $(HUB_IMAGE):$(VERSION) and $(HUB_IMAGE):latest"
	@echo "Next:  git tag -a v$(VERSION) -m \"v$(VERSION)\" && git push origin v$(VERSION)"

# ---- Housekeeping ----
clean:
	rm -rf bin/ dist/ static/app.css

help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
