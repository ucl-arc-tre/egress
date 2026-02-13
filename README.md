# egress

> [!WARNING]
> This repository is under development and is _not_ production ready.

UCL ARC TRE egress service is an internal API which provides a layer
on top of a storage backend to track file approvals prior to download.

See Egress service [system context](./docs/context/README.md) for an overview
of how this service interracts with other systems and actors.

## Installation

Install using [helm](https://helm.sh/)

```bash
helm install egress oci://ghcr.io/ucl-arc-tre/charts/egress \
  --version 0.3.0
```

See [chart/values.yaml](./chart/values.yaml) for values.

## Architecture

See the documentation on [component architecture](./docs/design/architecture.md) and [operation flows](./docs/design/operations.md) for details.
