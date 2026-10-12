# Security Policy

## Supported versions

Security fixes go into the latest release. Upgrade to it to get them; older releases are not
patched.

| Version | Supported |
| ------- | --------- |
| Latest release | :white_check_mark: |
| Older releases | :x: |

## Reporting a vulnerability

**Please do not report security vulnerabilities in public issues, pull requests or discussions.**

Report them privately through GitHub:
[**Report a vulnerability**](https://github.com/fbsobreira/gotron-sdk/security/advisories/new)
(the repository's **Security** tab, then **Report a vulnerability**). Only the maintainers can see
the report.

Please include:

- the affected package, command or file, and the version or commit;
- what an attacker can do, and under which conditions;
- steps or code to reproduce it;
- a suggested fix or mitigation, if you have one.

## In scope

- The SDK packages under `pkg/` (for example key handling, signing, keystores, address and amount
  parsing, and the gRPC client's handling of node responses).
- The `tronctl` CLI, its release archives (`tronctl_*` and the FN-DSA-512 `tronctl-pq_*` builds)
  and the installer (`install.sh`, served at `gotron.sh/install`).
- The build and release workflows in `.github/workflows/`.

A vulnerability in a dependency (for example gRPC, `golang.org/x/crypto` or liboqs) should be
reported to that project. Tell us too if it affects GoTRON, so we can update the dependency and
release a fix.

## What to expect

- We acknowledge your report within 3 business days.
- We send a first assessment within 10 business days: whether we accept the report, how severe we
  think it is, and the next steps.
- We keep you informed while we work on a fix, and tell you when it is released.

## Disclosure

We follow coordinated disclosure:

1. We work on the fix in a private fork or a GitHub security advisory draft, and may invite you to
   review it.
2. We release the fix, then publish a
   [GitHub security advisory](https://github.com/fbsobreira/gotron-sdk/security/advisories) with
   the affected and fixed versions, and request a CVE when the issue warrants one.
3. We credit you in the advisory, unless you prefer not to be named.

Please give us a reasonable time to release a fix before disclosing the issue publicly. We aim to
fix accepted reports within 90 days and will agree a date with you if it takes longer.

Thank you for helping keep GoTRON and its users safe.
