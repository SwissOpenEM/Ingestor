# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Removed

### Fixed

## v1.0.0-beta.5 (2025-08-19)

### Added

- Add directory-based access control via `.ingestor-access.yaml` files
- (Config) `Add WebServer.Other.GlobalConcurrencyLimit` to configure the number of concurrent tasks
- (Config) Add `Transfer.Method: None` option if data is mounted centrally (#150)
- (Config) Add `Webserver.Other.SecureCookies` option. This should be true when serving HTTPS from behind a reverse proxy (otherwise the default should set it correctly for either HTTP or HTTPS).
- (Config) Add `INGESTOR_SERVICE_USER_NAME` and `INGESTOR_SERVICE_USER_PASS` env variables

### Fixed

- Set cookies to 'SameSite: none' for cross origin requests (#153)

## v1.0.0-beta.4 (2025-06-17)

### Fixed

- Fixes a race condition in SSE communication (#148)
- Additional bug fixes for refreshing S3 tokens (#149)

## v1.0.0-beta.3 (2025-06-10)

### Added

- Support refreshing of S3 tokens (#140)

### Fixed

- Fix compatibility with Scicat API v3 (#137)
- Authentication and authorization fixes (#146)

## v1.0.0-beta.2 (2025-05-15)

### Added

- Configurable, templated destination path for datasets when using Globus for transfer (#101)
- (CONFIG) Transfer.Globus.DestinationTemplate contains a template for determining the path of the dataset on the destination side (#101)
- (Config) Transfer.ExtGlobus.CollectionRootPath is the path at which the Globus Collection is rooted. This value is used to convert absolute paths to Globus paths (#126)
- (CONFIG) Transfer.StorageLocation sets an ID string in the dataset lifecycle that specifies to which facility we're transmitting the dataset's data (#127)

### Changed

- (CONFIG) Transfer.Globus.SourceCollection becomes Transfer.Globus.SourceCollectionID (no change in expected value) (#101)
- (CONFIG) Transfer.Globus.DestinationCollection becomes Transfer.Globus.DestinationCollectionID (no change in expected value) (#101)
- (CODE) Some refactoring, particularly there's now separate S3 and Globus add task functions (#101)
- (Config) Webserver.Paths.CollectionLocation is now WebServer.Paths.CollectionLocations (plural), and its contents are now a map that maps collection location names (strings) to paths (strings). (#126)
- (Code) the API now expects dataset paths to have their first path node to contain the collection location 'name'. (#126)

A recommendation is to have the name of the collection's root folder to be the same as its "name" in the map. For example,  "example_collection" is  mapped to "/some/location/example_collection".

- (Config) Transfer.Globus.SourcePrefixPath was changed to Transfer.Globus.CollectionRootPath due to the change regarding multiple collection locations. (#135)

This changed value **does not** function in the same way: instead of adding a prefix path to the path given as SourceFolder of the dataset, it is now applied after the collection location resolution, making the absolute path relative to the root of the Globus collection.

Basically, if `location1` is a collection location that points to `/datasets/locations/location1`, and the user gives "/location1/dataset1" as a dataset `SourceFolder`, then the ingestor will first transform it to `/datasets/locations/location1/dataset1`.

Then, if the `CollectionRootPath` is set to `/datasets/locations`, then that means the Globus collection's root path is mounted there in the filesystem, and the Ingestor will transform the path in a way that removes the upper levels of the path, resulting in `/location1/dataset1` as the source path when requesting a transfer from Globus.

### Removed

- (CONFIG) Transfer.Globus.DestinationPrefixPath is removed because DestinationTemplate replaces it in functionality (#101)

## v1.0.0-beta.1 (2025-02-11)

Test release for the upcoming OpenEM workshop. Beta releases correspond to "Milestone V: Beta Release"

### Added

- Feature/extractor task pool (#48)
- Feature/redirect to frontend (#52)
- add collection location to sourcefolder path (#57)
- Feature/health check (#55)
- add userinfo endpoint (#76)
- Feature/scicat token (#77)
- Add docker file and docker compose (#79)

### Changed

- Refactor/use mainline scicatcli (#50)
- update default config (#56)
- ci: use ubuntu--22.04 as build image (#64)
- CI: add windows service build and release (#74)

### Fixed

- add the frontend redirect to the callback endpoint as well (#58)
- Ingest from collection location (#61)
- Bugfix/get datasets auth (#62)
- Feature/csrf safe cookie auth (#63)
- 'GET /metadata' - use params instead of request body (#75)

## v1.0.0-alpha.3 (2024-12-17)

### Added

- Add missing README to the package

## v1.0.0-alpha.2 (2024-12-17)

### Added

- Add `/extractor` endpoints (#43)
- Add OIDC auth (#41)

### Fixed

- Fix default config file
- CI fixes
- Config file parsing fixes (#39, #47)

## v1.0.0-alpha.1 (2024-12-16)

Alpha version for internal testing. Alpha releases correspond to "Milestone IV: Alpha Release"

## v0.1.0 (2024-11-25)

Initial version

## v0.0.1 -- v0.0.11 (2024-11-20)

Tests of the release process and continuous integration
