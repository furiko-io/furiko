# Development Guide

This document aims to provide tips and guidelines for developing and working on Furiko.

## Development Environment

### Setting up a Kubernetes Cluster

To start developing on Furiko, you will need to set up a local development environment with a Kubernetes cluster for testing purposes. There are several ways to set one up if you don't already have one:

1. [**kind**](https://kind.sigs.k8s.io/) (recommended): Uses Docker containers to run a full control plane and nodes locally on Linux, macOS and Windows.
2. [**minikube**](https://minikube.sigs.k8s.io/docs/): Sets up a Kubernetes cluster locally using a VM.
3. [**kubeadm**](https://kubernetes.io/docs/reference/setup-tools/kubeadm/): Sets up a full Kubernetes cluster, like in a production environment.

There are many other ways of deploying and setting up a Kubernetes cluster not covered here, and they may also work for development purposes. Most of Furiko development is sufficient to be run on KIND.

### Setting up a Go Environment

Furiko is written in Go, and thus requires a Go development environment. Furiko uses [Go Modules](https://go.dev/blog/using-go-modules), and thus expects a Go version of 1.13 or later. It is recommended to use a Go version that matches the version that Furiko uses for deployment, which can be found in the [`go.mod`](./go.mod) file in the repository.

For a tutorial on how to start writing Go, you can use [Go by Example](https://gobyexample.com/).

## Navigating the Project

### Monorepo Structure

`furiko-io/furiko` is intended to be structured as a [monorepo](https://monorepo.tools/). Although it is currently in its infancy, we expect this repository quickly grow to house multiple components, including Execution, Federation and Telemetry. This is to aid and reduce the toil needed when sharing Go code across multiple repositories, as well as to consolidate tooling, code style and development workflows when working with many components in a large project. This approach can be seen in other Go projects by the likes of [`kubernetes/kubernetes`](https://github.com/kubernetes/kubernetes), [`projectcalico/calico`](https://github.com/projectcalico/calico), and so on.

Notably, releases have to be done for all components in the single monorepo simultaneously. We believe that this is in fact _a good thing_, and users _should_ be running the same version of all components to ensure maximum compatibility and unhandled bugs arising from version mismatch.

There is [no silver bullet](https://en.wikipedia.org/wiki/The_Mythical_Man-Month), and several challenges may arise when using a monorepo approach. We may also consider splitting up the monorepo in the future.

Components that are considered outside the scope of the Furiko monorepo include documentation, web user interfaces, extensions not officially supported by Furiko, etc.

### Repository Layout

The following aims to distill and explain the high-level layout of this repository. Several conventions are adopted from [Kubebuilder](https://kubebuilder.io/), which is used sparsely in this project.

- `apis`: Contains GroupVersionKind definitions for types used in Furiko. They may or may not be `CustomResourceDefinition`s.
  - The convention is to add the type in `apis/<group>/<version>/<kind>_types.go`.
  - To add a new API, refer to [_Code Generation_](#code-generation) below.
- `cmd`: Contains the entrypoints for the project.
  - An entrypoint refers to a single, standalone binary built with `go build`.
  - Each entrypoint has to have a dedicated release target via CI.
- `config`: Contains YAML manifests.
  - Following Kubebuilder conventions, most of this directory uses [Kustomize](https://kustomize.io/) to build the final config, starting from `config/default`.
- `hack`: Contains auxiliary scripts, tooling, artifacts, etc. needed in this repository.
- `pkg`: Contains the bulk of Go code in the project. Code should be added to one of the following top-level packages in this directory, but one may add a new top-level package if it is reasonable to do so. The following list explains some of them.
  - `config`: Contains statically defined configuration for the project. Currently, it defines configuration for the `DefaultLoader`, which is used in the absence of [dynamic configuration](https://furiko.io/reference/configuration/dynamic/).
  - `core`: Contains core logic. What is defined as "core logic" should be sufficiently complex behavior, that is not specific to Kubernetes (i.e. it is not controller-specific code). For example, evaluating of Job Options is sufficiently complex and may be used in multiple components (e.g. CLI tools), and thus should be housed in `core`.
  - `execution`: Package for the Execution component. This package serves as a guiding example for how component packages should be further structured:
    - `controllers`: Controllers for the `execution-controller`.
    - `mutation`: Mutation logic for `execution-webhook`.
    - `validation`: Validation logic for `execution-webhook`.
    - `util`: Utility functions specific to Execution. If it may be used in other components, consider putting it in the upper `pkg/util` package instead.
  - `generated`: Contains generated code, do not edit. See [_Code Generation_](#code-generation) below.
  - `runtime`: Contains shared libraries for controller (and webhook) runtime. The analog in Kubebuilder is [`controller-runtime`](https://github.com/kubernetes-sigs/controller-runtime), which this is loosely based on.
  - `util`/`utils`: Contains shared utilities across Furiko.

Conforming to the above project structure keeps code where we expect, and improves discovery and readability of code. That said, these guidelines only serves to act as a recommendation, and may be broken or amended any time if deemed reasonable.

## Running the Code Locally

First, ensure that you have a Kubernetes cluster that is working and contactable from your local machine (refer to [_Setting up a Kubernetes Cluster_](#setting-up-a-kubernetes-cluster)).

Create a local, uncommitted copy of the [_Bootstrap Configuration file_](https://furiko.io/reference/configuration/bootstrap/) for the component you want to run. For example, the config file used for `execution-controller` is in `config/execution/controller/config.yaml`. One way is to create it in `.local`, which will be ignored by version control in this project.

Finally, start the program and pass the relevant command-line flags:

```sh
# Use verbosity level 5
go run ./cmd/execution-controller \
  -config=.local/execution_controller_config.yaml \
  -v=5
```

## Code Generation

Furiko makes extensive use of code generation in the project. Instead of using `go generate`, code generation is managed via `Makefile`.

A succinct summary of all Makefile steps is available using `make help`.

### Generating Manifests

Some manifests in `config` are generated using [controller-gen](https://github.com/kubernetes-sigs/controller-tools), including CustomResourceDefinition, WebhookConfiguration and RBAC resources.

Code generation requires the existence of `+kubebuilder` markers. For more information, refer to the [Kubebuilder documentation](https://kubebuilder.io/reference/markers.html).

To regenerate all manifests, run the following:

```sh
make manifests
```

### Generating Go Code

Furiko also uses generated clientsets, informers and listers using `generate-groups.sh`, a script in a project also known as [`code-generator`](https://github.com/kubernetes/code-generator).

In addition, `deepcopy-gen` is used to generate `DeepCopy` implementations.

To regenerate all Go code, run the following:

```sh
make generate
```

You can also run just one of the generation steps in the Makefile, they will be prefixed with `generate-`.

### Creating a new API

We do not use the process as described in the Kubebuilder book [here](https://kubebuilder.io/cronjob-tutorial/new-api.html), because we do not use Kubebuilder's framework or configuration management conventions anymore. Instead, simply copy existing API packages and modify them accordingly.

Some general pointers:

1. When adding a new GroupVersion, ensure that `doc.go` exists in the version package with a `+groupName` marker. The file name MUST be `doc.go` otherwise `generate-groups.sh` will use the incorrect group.
2. The convention is to add a `register.go` file in the Group package, for containing references that don't need the Version.
3. Follow the conventions in the `config` folder to register new Deployment/Service manifests.

### Code Generation Lint

It is also imperative that generated code should always be up-to-date with the markers from which it is generated from in a single commit. To enforce this, we also have a CI step called `generate-lint` which re-runs all code generation and fails if some differences are detected on each commit.

## Code Style

We employ `go fmt` and `goimports` applied to all Go code, including generated code. We also use [golangci-lint](https://golangci-lint.run/), which checks for common stylistic issues or possible errors.

## Testing

### Unit Tests

As much as possible, write unit tests to verify the correctness of the changes that you make. We use [Codecov](https://codecov.io/) to track and visualize code coverage over time.

### Testing Locally

Some complex changes or addition of new features may require testing that goes beyond unit tests that we can write. In such a case, please run your code locally and demonstrate that it works with a Kubernetes cluster, such as via a screenshot, video capture, or logs in your pull request.

See [Running the Code Locally](#running-the-code-locally) for more information.

### Testing Against a Development Cluster

If you want to test with your code actually running in the Kubernetes cluster (e.g. for testing interaction with webhooks), you will need to be able to package your local changes as a container image, and be able to pull them in your testing cluster.

If you are using `kind`, you can refer to their guide on [how to set up a local registry](https://kind.sigs.k8s.io/docs/user/local-registry/).

The `Makefile` supports building and pushing Docker images to a local registry, and deploying all components to a configured Kubernetes cluster for your convenience. Assuming you have a working Kubeconfig and have set up a local registry at `localhost:5000`, you can build, push and deploy all components with one line:

```sh
make dev
```

Configurable options via environment variables:

- `DEV_IMAGE_TAG`: The image tag to be built and deployed. Defaults to `dev`.
- `DEV_IMAGE_REGISTRY`: The image registry to push to. Defaults to `localhost:5000`.
- `DEV_IMAGE_NAME_PREFIX`: The image name prefix to push to and use. Defaults to `$DEV_IMAGE_REGISTRY/furikoio`.

The above Makefile target consists of three sub-targets, which are run in order when using `make dev`:

- `make dev-build`: Builds Docker images for all components locally, with each image tagged as `$DEV_IMAGE_TAG`.
- `make dev-push`: Pushes the locally built Docker images to `$DEV_IMAGE_REGISTRY`.
- `make dev-deploy`: Generates Kubernetes manifests for all Deployments to use `$DEV_IMAGE_TAG` from `$DEV_IMAGE_REGISTRY` and other YAMLs, and immediately applies them with `kubectl apply` to the currently configured Kubernetes cluster.

### Staging Environment

We currently don't have a staging enviroment for validating changes and performing regression/end-to-end tests. If you are experienced or wish to explore setting up of testing infra, we more than welcome your contributions :)

## Releases

### Official Releases

Releases are determined by Git tags, and will be published primarily via the [Releases page](https://github.com/furiko-io/furiko/releases).

Release automation is done using GitHub actions ([release.yml](https://github.com/furiko-io/furiko/blob/main/.github/workflows/release.yml)). To automate boring tasks, we use [GoReleaser](https://goreleaser.com/) to compile, package, push and publish releases when a new tag is pushed.

The release workflow automatically creates a draft release [here](https://github.com/furiko-io/furiko/releases). We expect release notes to contain both a high-level summary of changes since the last release, and a detailed changelog which should be generated using GoReleaser.

Container images are automatically pushed to Docker Hub under the [`furikoio` namespace](https://hub.docker.com/r/furikoio).

Finally, releases are minimally expected to contain the following artifacts, downloadable from the Releases page:

1. Binary executables (in `.tar.gz` archive format)
2. Combined YAML manifest, for installation [as described on the website](https://furiko.io/guide/setup/install/#from-yaml)

When we introduce additional components (i.e. Federation and Telemetry), release workflows (and this guide) may be further revised.

### Snapshot Releases

We also publish a snapshot of the current `HEAD` of the `main` branch, which will help facilitate testing and allow users to install the latest cutting-edge version if desired.

On every merge/commit to the `main` branch, we will push Docker images with the `snapshot` tags to denote these snapshot releases. Additional tags like `v0.1.2-next` also will be pushed, which denotes the next planned version's snapshot.

To install the current snapshot release via Kustomize, you can run the following:

```sh
IMAGE_TAG=snapshot make deploy
```

Other possible values for `IMAGE_TAG`:

- `latest` (default): Points to the current latest stable release of Furiko.
- `snapshot`: Points to the latest `HEAD` of Furiko (i.e. HEAD of `main`).
