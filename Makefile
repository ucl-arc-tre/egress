SHELL := /bin/bash
.PHONY: *

K3D_CLUSTER_NAME := "ucl-arc-tre-egress"
K3D_K3S_IMAGE_VERSION := rancher/k3s:v1.34.2-k3s1
DEV_NODEPORT := 30001
DEV_EXTERNAL_PORT := 8080
DEV_KUBECONFIG_PATH := "kubeconfig.yaml"
DEV_IMAGE := "localhost/ucl-arc-tre-egress:dev"
RELEASE_IMAGE := "localhost/ucl-arc-tre-egress:release"

define assert_command_exists
	if ! command -v $(1) &> /dev/null; then \
		echo -e "\033[0;31mERROR\033[0m: $(2)" && exit 1; \
	fi
endef

help: ## Show this help
	@echo
	@grep -E '^[a-zA-Z_0-9-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%s\033[0m|%s\n", $$1, $$2}' \
        | column -t -s '|'
	@echo

codegen:  ## Run code generation
	go generate ./...

test-unit:  ## Run unit tests
	go test ./internal/...

test-e2e: dev-k3d dev-rustfs ## Run end-to-end tests
	docker buildx build --tag $(RELEASE_IMAGE) --target release .
	k3d image import $(RELEASE_IMAGE) -c $(K3D_CLUSTER_NAME)
	helm upgrade --install --create-namespace -n e2e -f e2e/values.yaml egress ./chart
	go test ./e2e/...

dev: dev-requirements dev-k3d dev-rustfs ## Deploy dev env
	docker buildx build --tag $(DEV_IMAGE) --target dev .
	k3d image import $(DEV_IMAGE) -c $(K3D_CLUSTER_NAME)
	$(MAKE) dev-helm

dev-destroy: ## Destroy the dev env
	k3d cluster delete $(K3D_CLUSTER_NAME)

dev-helm: ## Deploy the dev helm chart
	helm upgrade --install --create-namespace -n dev -f deploy/dev/values.yaml egress ./chart

dev-k3d: ## Build a k3d cluster for dev, if it doesn't exist already
	if ! k3d cluster list | grep -q $(K3D_CLUSTER_NAME); then \
	  k3d cluster create $(K3D_CLUSTER_NAME) \
	    --image $(K3D_K3S_IMAGE_VERSION) \
		--api-port 6550 \
		--servers 1 \
		--agents 0 \
		--port "${DEV_EXTERNAL_PORT}:${DEV_NODEPORT}@server:0:direct" \
		--k3s-arg="--disable=traefik@server:*" \
		--k3s-arg="--disable=metrics-server@server:*" \
		--k3s-arg="--disable-cloud-controller@server:*" \
		--k3s-arg="--disable-helm-controller@server:*" \
		--k3s-arg="--etcd-disable-snapshots@server:*" \
		--volume "$${PWD}:/repo@all" \
		--no-lb \
		--wait; \
	fi
	k3d kubeconfig get $(K3D_CLUSTER_NAME) > $(DEV_KUBECONFIG_PATH)

dev-rustfs: ## Install rustfs as an S3 compatible object store
	helm repo add rustfs https://charts.rustfs.com
	helm upgrade rustfs rustfs/rustfs -n rustfs --create-namespace --install \
	  --set mode.standalone.enabled=true \
	  --set replicaCount=1 \
      --set mode.distributed.enabled=false

dev-requirements:  ## Check if the dev requirements are satisfied
	$(call assert_command_exists, go, "Please install go: https://go.dev/doc/install")
	$(call assert_command_exists, k3d, "Please install k3d: https://k3d.io/stable/#installation")
	$(call assert_command_exists, helm, "Please install helm: https://helm.sh/docs/intro/install/")

.SILENT:  # all targets
