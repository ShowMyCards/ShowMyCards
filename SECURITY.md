# Security Policy

ShowMyCards is a self-hosted application. There is intentionally no
authentication, authorisation, or **inbound** rate limiting on the backend
API — it is designed to be run locally or behind a trusted reverse proxy on
a network the operator controls.

**Outbound** rate limiting toward third-party services (notably the Scryfall
API) is a separate concern: we are obliged to respect upstream rate limits,
and the codebase should always do so.

## Reporting a Vulnerability

If you believe you have found a security issue, **please do not open a public
issue**. Instead, report it privately via GitHub's
[private vulnerability reporting](https://github.com/ShowMyCards/ShowMyCards/security/advisories/new).

When reporting, include:

- A description of the issue and its impact.
- Steps to reproduce, ideally with a minimal repro.
- The version (Docker tag, commit SHA, or release tag) you tested against.
- Any suggested mitigation, if you have one.

You can expect an acknowledgement within 7 days. Once a fix is available, we
will coordinate disclosure and credit you in the release notes unless you
prefer to remain anonymous.

## Supported Versions

Only the most recent released version receives security fixes. The `latest`
Docker tag tracks the most recent release. Tracked release lines are visible
on the [GitHub Releases page](https://github.com/ShowMyCards/ShowMyCards/releases).

## Out of Scope

The following are by design and will not be treated as vulnerabilities:

- Lack of authentication, authorisation, or inbound rate limiting on the
  backend API. (Outbound rate limiting toward third-party APIs **is**
  in scope — failing to respect upstream limits is a bug.)
- The application being exploitable when exposed directly to the public
  internet without a reverse proxy or other access control.
- Information disclosure from data the operator has loaded into their own
  instance.
