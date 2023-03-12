projectname?=subst
K3S_NAME ?= subst-cmp

VERSION ?= $$(git describe --abbrev=0 --tags --match "v*")
IMG ?= ghcr.io/buttahtoast/subst:$(VERSION)
PLUGIN_IMG ?= ghcr.io/buttahtoast/subst-cmp:$(VERSION)

# Get information about git current status
GIT_HEAD_COMMIT ?= $$(git rev-parse --short HEAD)
GIT_TAG_COMMIT  ?= $$(git rev-parse --short $(VERSION))
GIT_MODIFIED_1  ?= $$(git diff $(GIT_HEAD_COMMIT) $(GIT_TAG_COMMIT) --quiet && echo "" || echo ".dev")
GIT_MODIFIED_2  ?= $$(git diff --quiet && echo "" || echo ".dirty")
GIT_MODIFIED    ?= $$(echo "$(GIT_MODIFIED_1)$(GIT_MODIFIED_2)")
GIT_REPO        ?= $$(git config --get remote.origin.url)
BUILD_DATE      ?= $$(git log -1 --format="%at" | xargs -I{} date -d @{} +%Y-%m-%dT%H:%M:%S)

default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build golang binary
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -o $(projectname)

.PHONY: install
install: ## install golang binary
	@go install -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"

.PHONY: run
run: ## run the app
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"  main.go

.PHONY: bootstrap
bootstrap: ## install build deps
	go generate -tags tools tools/tools.go

PHONY: test
test: clean ## display test coverage
	go test -json -v ./... | gotestfmt

PHONY: clean
clean: ## clean up environment
	@rm -rf coverage.out dist/ $(projectname)

PHONY: cover
cover: ## display test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

PHONY: fmt
fmt: ## format go files
	gofumpt -w .
	gci write .

PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml

.PHONY: docker-build
docker-build: ## dockerize golang application
	docker build . -f Dockerfile -t ${IMG} --build-arg GIT_HEAD_COMMIT=$(GIT_HEAD_COMMIT) \
 		--build-arg GIT_TAG_COMMIT=$(GIT_TAG_COMMIT) \
 		--build-arg GIT_MODIFIED=$(GIT_MODIFIED) \
 		--build-arg GIT_REPO=$(GIT_REPO) \
 		--build-arg GIT_LAST_TAG=$(VERSION) \
 		--build-arg BUILD_DATE=$(BUILD_DATE)

.PHONY: docker-build-cmp
docker-build-cmp: ## build argocd plugin
	docker build . -f argocd-cmp/Dockerfile -t ${PLUGIN_IMG} --build-arg GIT_HEAD_COMMIT=$(GIT_HEAD_COMMIT) \
 		--build-arg GIT_TAG_COMMIT=$(GIT_TAG_COMMIT) \
 		--build-arg GIT_MODIFIED=$(GIT_MODIFIED) \
 		--build-arg GIT_REPO=$(GIT_REPO) \
 		--build-arg GIT_LAST_TAG=$(VERSION) \
 		--build-arg BUILD_DATE=$(BUILD_DATE) 

.PHONY: pre-commit
pre-commit:	## run pre-commit hooks
	pre-commit run --all-files

kind-up:
	@echo "Building kubernetes $${KIND_K8S_VERSION:-v1.25.0}..."
	@kind create cluster --name $(K3S_NAME) --image kindest/node:$${KIND_K8S_VERSION:-v1.25.0} --wait=120s 

.PHONY: kind-down
kind-down:
	@echo "Deleting cluser..."
	@kind delete cluster --name $(K3S_NAME)

# Local ArgoCD Development environment
.PHONY: argocd-dev
argocd-dev: kind-up argocd-install load-cmp
	@echo "ArgoCD is available at http://localhost:8080"

.PHONY: argocd-install
argocd-install:
	@helm repo add argo https://argoproj.github.io/argo-helm
	@helm repo update
	@helm upgrade --install argo/argocd --namespace argocd --create-namespace --name argocd --values argocd-values.yaml

# Loads local cmp build of subst into kind cluster and restarts argocd components
.PHONY: load-cmp
load-cmp:
	@make docker-build-cmp
	@kind load docker-image ${PLUGIN_IMG} --name $(K3S_NAME)
