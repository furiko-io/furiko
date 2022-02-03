# Define main package name
PKG = github.com/furiko-io/furiko

# Define the stable version of the Kubernetes API we should build for.
# Changing this may produce slightly different generated client code.
KUBE_VERSION = "v0.23.0"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# GENERATE_GROUPS_GENERATORS defines which generators should be run using generate-groups.
GENERATE_GROUPS_GENERATORS ?= "client,lister,informer"

# GENERATE_GROUPS_FLAGS defines flags to be passed to generate-groups.sh.
# To produce more debug output, set GENERATE_GROUPS_FLAGS="--v=2"
GENERATE_GROUPS_FLAGS ?= --v=1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: manifests generate fmt

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

# Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
.PHONY: manifests
manifests: tidy controller-gen yq
	# Generate CRDs
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=config/crd/bases
	# Generate webhook manifests
	$(CONTROLLER_GEN) webhook paths="./apis/execution/..." output:dir=config/common/webhook/execution
	# Generate ClusterRole manifests
	$(CONTROLLER_GEN) rbac:roleName=controller-role paths="./cmd/execution-controller/..." output:stdout > config/common/rbac/execution/controller/cluster_role.yaml
	$(CONTROLLER_GEN) rbac:roleName=webhook-role paths="./cmd/execution-webhook/..." output:stdout > config/common/rbac/execution/webhook/cluster_role.yaml
	# Add preserveUnknownFields manually with yq, see https://github.com/kubernetes-sigs/controller-tools/issues/476
	ls -1 config/crd/bases/*.yaml | xargs -I {} $(YQ) e '.spec.preserveUnknownFields = false' -i {}

# Generate Go code.
.PHONY: generate
generate: generate-deepcopy generate-groups

# Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
generate-deepcopy: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Generate code such as client, lister, informers.
# TODO(irvinlim): Current version of generate-groups.sh requires running in GOPATH to generate correctly.
generate-groups: generate-groups.sh
	$(GENERATE_GROUPS) $(GENERATE_GROUPS_GENERATORS) "$(PKG)/pkg/generated" "$(PKG)/apis" execution:v1alpha1 --go-header-file=hack/boilerplate.go.txt $(GENERATE_GROUPS_FLAGS)

# Format code.
.PHONY: fmt
fmt: goimports
	go fmt ./...
	$(GOIMPORTS) -w -local "$(PKG)" .

# Lint all code.
.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run -v --timeout=5m

# Go mod tidy.
.PHONY: tidy
tidy:
	go mod tidy

# Run tests.
.PHONY: test
test: manifests generate fmt envtest
	./hack/run-tests.sh

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

GOIMPORTS = $(shell pwd)/bin/goimports
.PHONY: goimports
goimports: ## Download yq locally if necessary.
	$(call go-get-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports@latest)

YQ = $(shell pwd)/bin/yq
.PHONY: yq
yq: ## Download yq locally if necessary.
	$(call go-get-tool,$(YQ),github.com/mikefarah/yq/v4@v4.14.1)

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint@v1.44.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# generate-groups.sh will download generate-groups.sh which is used for generating client libraries.
generate-groups.sh:
	@{ \
	set -e ;\
	cd /tmp ;\
	rm -rf code-generator ;\
	git clone https://github.com/kubernetes/code-generator.git --branch $(KUBE_VERSION) ;\
	}
GENERATE_GROUPS=/tmp/code-generator/generate-groups.sh
