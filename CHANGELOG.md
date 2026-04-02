# Changelog

## v1.2.0

### Changed

- All CLI parameter checks and config-file semantic validation now complete before `netmap` initializes the Compute provider, checks ADC validity, or talks to the Compute API.
- The CLI usage/help text now comes from the user-editable embedded file [internal/app/usage.txt](/home/macho_prawn/gh-repo/netmap/internal/app/usage.txt), so help copy can be updated without editing the Go source that renders it.
- `netmap version` now prints `v1.2.0`.
- README and release documentation now reference `v1.2.0`, while the repository `VERSION` file remains the raw `1.2.0` source of truth used to derive release tag/title `v1.2.0`.
