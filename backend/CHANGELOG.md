# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
None yet.

## [1.0.3] - 2020-08-09
### Added
- Added grpc Time-To-Live save and get func

## [1.0.2] - 2020-07-25
### Added
- backend/main.go
  - added TLS secure connections from grpc client (frontend-go)
### Removed
- TLS pub/priv keys

## [1.0.1] - 2020-07-15
### Edited
- backend/data/redis.go
  - edited generate() function to detect collisions with set limits on length of code and # of code generation attempts.  Function will return a warning/error when limits are hit.

## [1.0.0] - 2020-07-14
### Added
- Initial build