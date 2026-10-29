# thdwb Support Process

This document describes the support process for thdwb, including how users
can get help, how bugs are tracked, and how the project maintains its FAQ and
documentation.

## Support Channels

| Channel | Purpose | Link |
|---------|---------|------|
| GitHub Issues | Bug reports, feature requests | https://github.com/danfragoso/thdwb/issues |
| GitHub Discussions | Questions, general discussion | https://github.com/danfragoso/thdwb/discussions |
| Email (support) | General support inquiries | support@thdwb.dev |
| Email (security) | Security vulnerability reports | security@thdwb.dev |
| Email (conduct) | Code of Conduct reports | conduct@thdwb.dev |

## Bug Triage Process

### 1. Issue Filed

A user files a bug report using the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md).

### 2. Triage

A maintainer triages the issue within 5 business days:

- Assigns labels (bug, enhancement, priority, etc.)
- Confirms the issue is reproducible
- Requests additional information if needed

### 3. Assignment

The issue is assigned to a maintainer or contributor for resolution.

### 4. Resolution

The fix is implemented, tested, and merged via pull request.

### 5. Closure

The issue is closed with a reference to the resolving pull request.

## Feature Request Process

### 1. Discussion

Feature requests are discussed in GitHub Discussions or as issues using the
[feature request template](.github/ISSUE_TEMPLATE/feature_request.md).

### 2. Design

Accepted features are designed with input from the community. Design
documents may be created in the repository.

### 3. Implementation

The feature is implemented in a phase branch (per PLAN.md) and reviewed via
pull request.

### 4. Documentation

User-facing features include documentation updates (user guide, API reference,
FAQ).

## FAQ Maintenance

The FAQ ([docs/support/FAQ.md](FAQ.md)) is maintained by the community:

- New questions are added when they appear frequently in issues/discussions.
- Answers are updated when behavior changes between releases.
- Outdated entries are removed or marked as deprecated.

## Release Support

Each release is supported until the next minor release. Security patches may
be backported to the latest minor release on a case-by-case basis.

## Response Time Targets

| Priority | Initial Response | Resolution Target |
|----------|-----------------|-------------------|
| Critical (security, data loss) | 5 business days | 30 days |
| High (major feature broken) | 10 business days | 60 days |
| Medium (minor bug) | 15 business days | Next release |
| Low (cosmetic, enhancement) | Next release cycle | As time permits |

These are best-effort targets for a community-driven project.

## Contact

- General: support@thdwb.dev
- Security: security@thdwb.dev
- Conduct: conduct@thdwb.dev
