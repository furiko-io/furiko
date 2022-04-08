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

# Set license header files.
LICENSE_HEADER_GO ?= hack/boilerplate.go.txt

# Set image name prefix. The actual image name and tag will be appended to this.
IMAGE_NAME_PREFIX ?= "docker.io/furikoio"

# Define the image tag to use, otherwise will default to latest.
# The latest tag always refers to the latest stable release.
IMAGE_TAG ?= "latest"

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

.PHONY: all
all: manifests generate fmt build yaml ## Generate code, build Go binaries and YAML manifests.

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
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: generate
generate: generate-deepcopy generate-groups ## Generate Go code.

generate-deepcopy: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="$(LICENSE_HEADER_GO)" paths="./..."

generate-groups: generate-groups.sh ## Generate code such as client, lister, informers.
	# TODO(irvinlim): Current version of generate-groups.sh requires running in GOPATH to generate correctly.
	$(GENERATE_GROUPS) $(GENERATE_GROUPS_GENERATORS) "$(PKG)/pkg/generated" "$(PKG)/apis" execution:v1alpha1 --go-header-file=$(LICENSE_HEADER_GO) $(GENERATE_GROUPS_FLAGS)

.PHONY: fmt
fmt: goimports ## Format code.
	go fmt ./...
	$(GOIMPORTS) -w -local "$(PKG)" .

.PHONY: lint
lint: lint-go lint-license ## Lint all code.

.PHONY: lint-license
lint-license: license-header-checker ## Check license headers.
	$(LICENSE_HEADER_CHECKER) "$(LICENSE_HEADER_GO)" . go

.PHONY: lint-go
lint-go: golangci-lint ## Lint Go code.
	$(GOLANGCI_LINT) run -v --timeout=5m

.PHONY: tidy
tidy: ## Run go mod tidy.
	go mod tidy

.PHONY: test
test: ## Run tests with coverage. Outputs to combined.cov.
	./hack/run-tests.sh

##@ YAML Configuration

## Location to write YAMLs to
YAML_DEST ?= $(shell pwd)/yamls
$(YAML_DEST): ## Ensure that the directory exists
	mkdir -p $(YAML_DEST)

KUSTOMIZE_DEST ?= $(shell pwd)/dist/kustomize
$(KUSTOMIZE_DEST):
	mkdir -p $(KUSTOMIZE_DEST)

.PHONY: yaml
yaml: yaml-execution ## Build kustomize configs. Outputs to dist folder.

.PHONY: yaml-execution
yaml-execution: manifests kustomize $(YAML_DEST) $(KUSTOMIZE_DEST) ## Build furiko-execution.yaml with Kustomize.
	DEST_DIR=$(KUSTOMIZE_DEST) ./hack/generate-kustomization.sh "$(IMAGE_NAME_PREFIX)" "$(IMAGE_TAG)"
	$(KUSTOMIZE) build $(KUSTOMIZE_DEST) -o $(YAML_DEST)/furiko-execution.yaml

.PHONY: manifests
manifests: tidy controller-gen yq ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects into the "config" directory.
	# Generate CRDs
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=config/crd/bases
	# Generate webhook manifests
	$(CONTROLLER_GEN) webhook paths="./apis/execution/..." output:dir=config/common/webhook/execution
	# Generate ClusterRole manifests
	$(CONTROLLER_GEN) rbac:roleName=controller-role paths="./cmd/execution-controller/..." output:stdout > config/common/rbac/execution/controller/cluster_role.yaml
	$(CONTROLLER_GEN) rbac:roleName=webhook-role paths="./cmd/execution-webhook/..." output:stdout > config/common/rbac/execution/webhook/cluster_role.yaml
	# Add preserveUnknownFields manually with yq, see https://github.com/kubernetes-sigs/controller-tools/issues/476
	ls -1 config/crd/bases/*.yaml | xargs -I {} $(YQ) e '.spec.preserveUnknownFields = false' -i {}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Installs the CRDs into the K8s cluster.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize $(KUSTOMIZE_DEST) ## Deploys snapshot version of all components to the K8s cluster. Useful for testing current HEAD.
	DEST_DIR=$(KUSTOMIZE_DEST) ./hack/generate-kustomization.sh "$(IMAGE_NAME_PREFIX)" "snapshot"
	$(KUSTOMIZE) build $(KUSTOMIZE_DEST) | kubectl apply -f -

.PHONY: undeploy
undeploy: $(KUSTOMIZE_DEST) ## Undeploy controller from the K8s cluster. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build $(KUSTOMIZE_DEST) | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Release

.PHONY: snapshot
snapshot: ## Release snapshot of current repository and upload Docker image.
	./hack/release-snapshot.sh

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Ensure that the directory exists
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOIMPORTS ?= $(LOCALBIN)/goimports
YQ ?= $(LOCALBIN)/yq
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
LICENSE_HEADER_CHECKER ?= $(LOCALBIN)/license-header-checker

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.8.0
YQ_VERSION ?= v4.14.1
GOLANGCILINT_VERSION ?= v1.45.2
LICENSEHEADERCHECKER_VERSION ?= v1.3.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN):
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE):
	@[ -f $(KUSTOMIZE) ] || curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST):
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: goimports
goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
$(GOIMPORTS):
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.
$(YQ):
	GOBIN=$(LOCALBIN) go install github.com/mikefarah/yq/v4@$(YQ_VERSION)

GOLANGCILINT_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT):
	@[ -f $(GOLANGCI_LINT) ] || curl -sSfL $(GOLANGCILINT_INSTALL_SCRIPT) | sh -s $(GOLANGCILINT_VERSION)

.PHONY: license-header-checker
license-header-checker: $(LICENSE_HEADER_CHECKER) ## Download license-header-checker locally if necessary.
$(LICENSE_HEADER_CHECKER):
	GOBIN=$(LOCALBIN) go install github.com/lsm-dev/license-header-checker/cmd/license-header-checker@$(LICENSEHEADERCHECKER_VERSION)

# generate-groups.sh will download generate-groups.sh which is used for generating client libraries.
generate-groups.sh:
	@{ \
	set -e ;\
	cd /tmp ;\
	rm -rf code-generator ;\
	git clone https://github.com/kubernetes/code-generator.git --branch $(KUBE_VERSION) ;\
	}
GENERATE_GROUPS=/tmp/code-generator/generate-groups.sh
