# ====================================================================================
# Setup Project
PROJECT_NAME := provider-matomo
PROJECT_REPO := github.com/maximilianbraun/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

GOTOOLCHAIN ?= go$(shell awk '$$1 == "go" { v = $$2; if (v ~ /^[0-9]+\.[0-9]+$$/) v = v ".0"; print v; exit }' go.mod)
export GOTOOLCHAIN

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
GOLANGCILINT_VERSION = 2.12.2
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

CROSSPLANE_CLI_VERSION ?= v2.3.4
-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

IMAGES = provider-matomo
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= ghcr.io/maximilianbraun
XPKG_REG_ORGS_NO_PROMOTE ?= ghcr.io/maximilianbraun
XPKGS = provider-matomo
-include build/makelib/xpkg.mk

# Force image building prior to xpkg build so the image is present in the daemon.
xpkg.build.provider-matomo: do.build.images

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

build.init: $(CROSSPLANE_CLI)

# Run the provider locally out-of-cluster.
run: go.build
	@$(INFO) Running Crossplane provider locally out-of-cluster . . .
	$(GO_OUT_DIR)/provider --debug

# ====================================================================================
# End-to-end (client-level) tests against a dockerized Matomo.
# Independent of the makelib integration-test machinery.

e2e-up:
	docker compose -f docker-compose.e2e.yml up -d

e2e-down:
	docker compose -f docker-compose.e2e.yml down -v

test-e2e: e2e-up
	set -a; eval "$$(bash hack/matomo-setup.sh | grep -E '^MATOMO_(URL|TOKEN|ADMIN_PASS)=')"; set +a; \
	$(GO) test -tags e2e ./internal/controller/... -run E2E -v

.PHONY: submodules fallthrough run e2e-up e2e-down test-e2e
