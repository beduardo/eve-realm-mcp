VERSION     = $(shell cat VERSION)
GIT_HASH   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

BINARY       := dist/eve-realm-mcp
MAIN_PKG     := ./cmd/eve-realm-mcp
VERSION_FILE := VERSION
DOCKER_IMAGE := k3d-eve-realm-registry.localhost:5100/eve-realm-mcp

.PHONY: build build-prod test test-all proto bump-patch bump-minor bump-major release-patch release-minor release-major docker-build docker-push deploy-local wait-rollout cluster-up cluster-down

build:
	mkdir -p dist
	go build -o $(BINARY) $(MAIN_PKG)

build-prod:
	mkdir -p dist
	go build -ldflags "-X main.Version=$(VERSION) -X main.GitHash=$(GIT_HASH) -X main.BuildDate=$(BUILD_DATE)" -o $(BINARY) $(MAIN_PKG)

test:
	go test -count=1 ./...

proto:
	mkdir -p gen/proto/mcp/v1
	protoc --proto_path=proto \
		--go_out=gen --go_opt=module=github.com/beduardo/eve-realm-mcp/gen \
		--go-grpc_out=gen --go-grpc_opt=module=github.com/beduardo/eve-realm-mcp/gen \
		mcp/v1/mcp.proto

bump-patch:
	@NEW_VERSION=$$(awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}' $(VERSION_FILE)); \
	echo "$$NEW_VERSION" > $(VERSION_FILE); \
	echo "Version bumped to $$NEW_VERSION"

bump-minor:
	@NEW_VERSION=$$(awk -F. '{printf "%d.%d.%d", $$1, $$2+1, 0}' $(VERSION_FILE)); \
	echo "$$NEW_VERSION" > $(VERSION_FILE); \
	echo "Version bumped to $$NEW_VERSION"

bump-major:
	@NEW_VERSION=$$(awk -F. '{printf "%d.%d.%d", $$1+1, 0, 0}' $(VERSION_FILE)); \
	echo "$$NEW_VERSION" > $(VERSION_FILE); \
	echo "Version bumped to $$NEW_VERSION"

release-patch: test bump-patch build-prod docker-build docker-push deploy-local wait-rollout

release-minor: test bump-minor build-prod docker-build docker-push deploy-local wait-rollout

release-major: test bump-major build-prod docker-build docker-push deploy-local wait-rollout

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE):$(VERSION) .

docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)

deploy-local:
	sed -e 's/VERSION_PLACEHOLDER/$(VERSION)/g' deploy/k8s/deployment.yaml | kubectl apply -f -
	sed -e 's/VERSION_PLACEHOLDER/$(VERSION)/g' deploy/k8s/service.yaml | kubectl apply -f -

wait-rollout:
	@true && kubectl rollout status deployment/eve-realm-mcp -n eve-realm --timeout=120s

cluster-up:
	bash deploy/k3d/setup.sh

cluster-down:
	bash deploy/k3d/teardown.sh

test-all: test
	bash deploy/k8s/tests/validate-deployment.sh
	bash deploy/k8s/tests/validate-service.sh
	bash deploy/k8s/tests/validate-deploy-local.sh
	bash deploy/k8s/tests/validate-wait-rollout.sh
	bash deploy/k8s/tests/validate-release-pipeline.sh
