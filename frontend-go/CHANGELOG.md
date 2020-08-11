# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
None yet.

## [1.0.3] - 2020-08-10
### Added
- Converted TTL from string to int64
- Added server/client Keepalive

## [1.0.2] - 2020-08-09
### Added
- Refactored Proto file for efficiency
  - added user choice of Time-To-Live
- Improved webgui

## [1.0.1] - 2020-07-25
### Added
- frontend/cmd/main.go
  - added TLS secure connection to grpc server
### Removed
- TLS pub key

## [1.0.0] - 2020-07-14
### Added
- Initial build