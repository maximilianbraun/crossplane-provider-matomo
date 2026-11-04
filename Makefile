PROVIDER = provider-matomo
BINARY = $(PROVIDER)
IMG ?= ghcr.io/maximilianbraun/$(PROVIDER):latest

GO := go
LOCALBIN ?= $(shell pwd)/bin
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.20.1
ANGRYJET_VERSION ?= v0.0.0-20251017183449-dd4517244339

.PHONY: all
all: generate build

.PHONY: generate
generate: controller-gen
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."
	"$(CONTROLLER_GEN)" crd paths="./apis/..." output:crd:artifacts:config=package/crds
	$(GO) run github.com/crossplane/crossplane-tools/cmd/angryjet@$(ANGRYJET_VERSION) generate-methodsets --header-file=hack/boilerplate.go.txt ./apis/...

.PHONY: manifests
manifests: generate

.PHONY: build
build:
	$(GO) build -o bin/$(BINARY) ./cmd/provider/

.PHONY: test
test:
	$(GO) test ./... -coverprofile cover.out

.PHONY: e2e-up
e2e-up:
	docker compose -f docker-compose.e2e.yml up -d

.PHONY: e2e-down
e2e-down:
	docker compose -f docker-compose.e2e.yml down -v

# test-e2e brings up Matomo, bootstraps it, and runs the e2e-tagged tests
# against it. Run `make e2e-down` afterwards to clean up.
.PHONY: test-e2e
test-e2e: e2e-up
	set -a; eval "$$(bash hack/matomo-setup.sh | grep -E '^MATOMO_(URL|TOKEN|ADMIN_PASS)=')"; set +a; \
	$(GO) test -tags e2e ./internal/controller/... -run E2E -v

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push:
	docker push $(IMG)

.PHONY: dev
dev: generate build
	./bin/$(BINARY) --debug

.PHONY: clean
clean:
	rm -rf bin/ package/crds/

$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN="$(LOCALBIN)" $(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
