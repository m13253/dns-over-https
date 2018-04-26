# Changelog

This Changelog records major changes between versions.

Not all changes are recorded. Please check git log for details.

## Version 1.3.5

- Limit the frequency of creating HTTP client on bad network condition

## Version 1.3.4

- doh-client now silently fails in case of network error to prevent caching of SERVFAIL
- EDNS0 is now inserted to the beginning of OPT section, to ensure DNSSEC signatures are at the end
- Improve building system
- Update documents

## Version 1.3.3

- Take User-Agent out of common library, that would be better for packaging

## Version 1.3.2

- Fix version string in HTTP User-Agent

## Version 1.3.1

- Fix the "address already in use" issue

## Version 1.3.0

- Breaking change: Add client / server support for multiple listen address
  The `listen` option in the configuration file is a list now

## Version 1.2.1

- Update protocol to IETF draft-07
- Update installation documentations for Ubuntu / Debian

## Version 1.2.0

- Add installation documentations for Ubuntu / Debian
- Include CloudFlare DOH server (1.1.1.1, 1.0.0.1) in default configuration
- Fix a problem causing `go get` to fail due to relative paths
- Add documentation about `/etc/hosts` preloading

## Version 1.1.4

- Add `no_cookies` option
- Add documentation on privacy issues
- Adapt for CloudFlare DNS service
- Fix a problem causing a single network failure blocking future requests
- Add experimental macOS support

## Version 1.1.3

- Unsupported Content-Type now generates HTTP error code 415

## Version 1.1.2

- Adapt to IETF protocol
- Optimize for HTTP caches

## Version 1.1.1

- Adapt to IETF protocol
- Optimize for HTTP caches
- Add documentation for uninstallation instructions
- Fix build issues

## Version 1.1.0

- Adpat to IETF protocol
- Fix issues regarding to HTTP caching
- Require Go 1.9 to build now
- Fix systemd issue

## Version 1.0.1

- Fix build issues

## Version 1.0.0

- First release
- Relicense as MIT license
