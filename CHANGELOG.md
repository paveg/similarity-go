# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.0] - 2025-09-19

### Added

- `CHANGELOG.md` to track notable changes following the Keep a Changelog format.
- Structured helpers for the weight benchmark CLI, making the reporting output
  reusable from tests and other tooling.

### Changed

- Refactored weight optimization reporting to share writer utilities and
  centralized constants for grid-search bounds and percentage calculations.
- Reworked genetic optimizer configuration with named constants and an explicit
  pseudo-random generator factory for deterministic tuning.

### Fixed

- Hardened similarity weight validation to ensure positive values, sensible
  tolerances, and documented penalty rules.
- Resolved lint, gosec, and vet warnings across optimization and benchmarking
  packages (config updater, statistical validator, and codebase benchmarks).
- Removed shadowed variables and updated file permissions in the config
  updater workflow to satisfy security linting.

## [v0.1.0] - 2025-08-28

### Added

- Initial public release of the similarity detection CLI with multi-factor
  AST analysis, directory scanning, and parallel processing support.
- Automated release packaging for Arch, Debian, RPM, macOS, Windows, and
  Android distributions via GitHub Actions.

### Fixed

- Addressed the initial set of lint issues and code scanning alerts prior to
  publishing the first release build.

---

[v0.2.0]: https://github.com/paveg/similarity-go/releases/tag/v0.2.0
[v0.1.0]: https://github.com/paveg/similarity-go/releases/tag/v0.1.0
[Unreleased]: https://github.com/paveg/similarity-go/compare/v0.2.0...HEAD
