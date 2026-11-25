# Contribution Guidelines

m13253/dns-over-https welcomes high-quality contributions from you!

## Pending tasks

High priority:
* Test each release
* Security hardening
* Software engineering improvements
* Modernize Go, fix bugs, and clear lint warnings

Medium priority:
* Better support for standalone server (certificate hot reload, OCSP stapling)
* Better HTTP connection pool management
* Support for HTTP/3 QUIC

Low priority:
* Refactoring selector modules
* Improving cross-software compatibility

Feel free to work on topics listed above or any topics you like!

## Release schedule

Versions are numbered `vX.Y.Z`. We bump `Z` for each backwards-compatible new version. We bump `X` or `Y` for major or minor incompatibilities. Dependency updates alone usually don’t lead to a new release, unless it contains relevant security fixes, functionality improvements, or it’s too long since the previous release.

We follow a simple and maybe even stupid release procedure. On each release:
1. We prepend itemized, notable changes since the last version to `Changelog.md`.
2. If we do need to bump `X` or `Y`, we edit `doh-{client,server}/version.go` to reflect the new version; otherwise, it should already contain the correct version number.
3. We commit the changes and tag the commit. That is the release commit.
4. After the release commit, we bump `Z` once, so the software that runs Git HEAD always reports a version number different from all previously released versions.

Bumping `Z` immediately after releasing seems unconventional, but not using suffixes like `-pre` or `-rc` simplifies version number comparison.

## Rules regarding Generative AI contributions

We don’t ask which tools you use. However, please ensure all your deliveries (code, comments, commit messages, issue reports, pull requests, etc.) are free from copyright concerns and **indistinguishable** from the output of a skilled human developer in terms of quality, coding style, and language style. In other words, it basically means that GenAI-assisted contributions must pass the Turing test.

While we appreciate all kinds of contributions, given the current status that no generative AI can pass the Turing test yet, please refrain from sending us fully-autonomous submissions.
