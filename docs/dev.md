# Development

The development environment uses a [k3d](https://k3d.io/stable/) kubernetes cluster, installing the chart and importing the image from local.
The image hot-reloads within the cluster by mounting the repo and running [air](https://github.com/air-verse/air).
Backend storage is provided by [rustfs](https://github.com/rustfs/rustfs), which is installed on `make dev`.
To show the available make commands for creating/destroying/updating the dev environment, at the root of this repo, run:

```bash
make help
```
