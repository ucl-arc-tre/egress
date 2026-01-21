# egress

> [!WARNING]
> This repository is under development and is _not_ production ready.

UCL ARC TRE egress service is an internal API which provides a layer
on top of a storage backend to track file approvals prior to download.

## Installation

```bash
helm install egress oci://ghcr.io/ucl-arc-tre/charts/egress \
  --version 0.2.0
```

## Architecture

### Database backends

- In memory (dev only): Beta
- [Rqlite](https://github.com/rqlite/rqlite): Planned
- [Postgres](https://github.com/postgres/postgres): Planned

### Storage backends

- S3: Beta
- Generic API: Planned
