# Contributing Guidelines

Thanks for contributing! This repository supports contributions from developers external to UCL. Because the project has access to potentially sensitive data in a Trusted Research Environment (TRE), we enforce extra security, privacy, and provenance controls. Please read this document before opening issues or pull requests.

1. Scope & expectations
- We use least-privilege, defense-in-depth, and provenance principles.

2. Legal & onboarding
<!--
- Sign the Contributor License Agreement (CLA).
-->
- Complete the short security & privacy orientation: /SECURITY_ORIENTATION.md
- By contributing you agree to the BSD 3-Clause License project license and to follow our Code of Conduct: CODE_OF_CONDUCT.md
- Enable [commit verification](https://docs.github.com/en/authentication/managing-commit-signature-verification) to ensure commit provenance.

3. Getting started
- Fork the repo and create a feature branch e.g. `feature1`.
- Run `pre-commit install` to install [pre-commit](https://pre-commit.com/).
- Run `make dev-requirements` to check all prerequisites are installed.

4. Branching & pull requests
- The `main` branch is protected. All direct pushes are disallowed.
- Open a Pull Request (PR) from your branch. PR title should be concise and reference any related issue e.g. `Fix: improve docs`.
- Fill out the PR template with a short summary and risk assessment.
- Include a migration plan or rollback steps if applicable.

6. Security & privacy requirements (must-haves)
- Automated checks:
  - Linting, unit tests, and end-to-end tests must pass.
  - Secrets scanning must report no secrets.
- Builds must be reproducible and produce signed artifacts.
- No runtime fetching of arbitrary external dependencies.
- Significant changes will be pen-tested.

8. Runtime
- The service runs in a container with a default deny network policy.
- Resource quotas: CPU, memory are set.
- No use of privileged syscalls or devices.

9. Review & merge policy
- All PRs require at least two maintainer reviews.
- Re-approvals are required for any change to a previously approved PR.
- PRs are blocked until all checks and required approvals are satisfied.

10. Disclosure & incident reporting
- If you find a vulnerability, follow our [private disclosure process](https://github.com/ucl-arc-tre/egress/blob/main/.github/SECURITY.md).

11. Support & contact
- Use the issue tracker for feature requests and non-sensitive bugs.
- For questions about contribution classification or running in the TRE, contact arc.tre[at]ucl.ac.uk.
