# Furiko

[![CI](https://github.com/furiko-io/furiko/actions/workflows/ci.yml/badge.svg)](https://github.com/furiko-io/furiko/actions/workflows/ci.yml) [![codecov](https://codecov.io/gh/furiko-io/furiko/branch/main/graph/badge.svg?token=ZSG05UXWJJ)](https://codecov.io/gh/furiko-io/furiko)

![Furiko Logo](./docs/images/color_horizontal.png)

**Furiko** is a cloud-native, enterprise-level cron and adhoc job platform for Kubernetes.

## Introduction

Furiko is a Kubernetes-native operator for managing, scheduling and executing scheduled and adhoc jobs and workflows. It aims to be an general-purpose job platform that supports a diverse range of use cases, including cron jobs, batch processing, workflow automation, etc.

Furiko is built from the beginning to support enterprise-level use cases and running self-hosted in a private Kubernetes cluster, supporting users across a large organization.

Some use cases that are perfect for Furiko include:

- Cron-based scheduling massive amounts of periodic jobs per day in a large organization
- Scheduling some jobs to run once at a later time, with a set of specific inputs
- Starting multiple jobs to execute one after another, once the previous job has finished
- Event-driven, offline/asynchronous job processing via webhooks
- Building a platform to automate business operations via form-based inputs (with Furiko as the job engine)

## License

**NOTE**: Although started within the company, Furiko is **not an official Shopee project or product**.

Furiko is licensed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0.txt).

Logo is designed by Duan Weiwei, and is distributed under [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/).
