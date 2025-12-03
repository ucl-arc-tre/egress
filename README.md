# egress

> [!WARNING]
> This repository is under development and is *not* production ready.

UCL ARC TRE egress service is an internal API which provides a layer
on top of a storage backend to track file approvals prior to download.

## Database backends

- In memory (dev only): In progress
- [Rqlite](https://github.com/rqlite/rqlite): Planned
- [Postgres](https://github.com/postgres/postgres): Planned

## Storage backends

- S3: In progress
- Generic API: Planned

## Contributing

Contributions are very welcome either. To suggest a change please:

- Fork this repository and create a branch.
- Run `pre-commit install` to install [pre-commit](https://pre-commit.com/).
- Modify, commit, push and open a pull request against `main` for review.
