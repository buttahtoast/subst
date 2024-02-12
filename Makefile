projectname?=subst
K3S_NAME ?= subst-cmp

# Get information about git current status
GOOS            ?= $(shell go env GOOS)
GOARCH          ?= $(shell go env GOARCH)
GIT_HEAD_COMMIT ?= $$(git rev-parse --short HEAD)
GIT_TAG_COMMIT  ?= $$(git rev-parse --short $(VERSION))
GIT_MODIFIED_1  ?= $$(git diff $(GIT_HEAD_COMMIT) $(GIT_TAG_COMMIT) --quiet && echo "" || echo ".dev")
GIT_MODIFIED_2  ?= $$(git diff --quiet && echo "" || echo ".dirty")
GIT_MODIFIED    ?= $$(echo "$(GIT_MODIFIED_1)$(GIT_MODIFIED_2)")
GIT_REPO        ?= $$(git config --get remote.origin.url)
BUILD_DATE      ?= $(shell git log -1 --format="%at" | xargs -I{} sh -c 'if [ "$(shell uname)" = "Darwin" ]; then date -r {} +%Y-%m-%dT%H:%M:%S; else date -d @{} +%Y-%m-%dT%H:%M:%S; fi')

# Docker Build
DOCKER_CLI_EXPERIMENTAL ?= enabled
LOCAL_PLATFORM := linux/$(GOARCH)
# Define platforms in cli if multiple needed linux/386,linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64,linux/ppc64le,linux/s390x 
TARGET_PLATFORMS ?= $(LOCAL_PLATFORM)

VERSION ?= $$(git describe --abbrev=0 --tags --match "v*")
IMG ?= ghcr.io/buttahtoast/subst:$(VERSION)
PLUGIN_IMG ?= ghcr.io/buttahtoast/subst-cmp:$(VERSION)

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
docker-build:
	@docker buildx create --use --name=cross --node=cross && \
	docker buildx build \
        --build-arg GIT_HEAD_COMMIT=$(GIT_HEAD_COMMIT) \
 		--build-arg GIT_TAG_COMMIT=$(GIT_TAG_COMMIT) \
 		--build-arg GIT_MODIFIED=$(GIT_MODIFIED) \
 		--build-arg GIT_REPO=$(GIT_REPO) \
 		--build-arg GIT_LAST_TAG=$(VERSION) \
 		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--platform $(TARGET_PLATFORMS) \
		--output "type=docker,push=false" \
		--tag $(IMG) \
		-f Dockerfile.argo-cmp \ 
		./



.PHONY: docker-build-cmp
docker-build-cmp: ## build argocd plugin
	docker build . -f Dockerfile.argo-cmp -t ${PLUGIN_IMG} --build-arg GIT_HEAD_COMMIT=$(GIT_HEAD_COMMIT) \
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

kind-load-image: PLUGIN_IMG = ghcr.io/buttahtoast/subst-cmp:local
kind-load-image: docker-build-cmp
	@echo "Loading image into cluster..."
	@kind load docker-image ${PLUGIN_IMG} --name $(K3S_NAME)

.PHONY: kind-down
kind-down:
	@echo "Deleting cluser..."
	@kind delete cluster --name $(K3S_NAME)

# Local ArgoCD Development environment
.PHONY: argocd-dev
argocd-dev: kind-up kind-load-image argocd-install
	@echo "ArgoCD is available at http://localhost:8080"
	make argocd-admin

.PHONY: argocd-install
argocd-install:
	@helm repo add argo https://argoproj.github.io/argo-helm
	@helm repo update
	@helm upgrade --install argocd argo/argo-cd --namespace argocd --create-namespace --values hack/argocd-values.yaml
	@kubectl kustomize hack/ | kubectl apply -f -

argocd-admin:
	@kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d