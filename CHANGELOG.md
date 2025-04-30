# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [PR #101] (2025-03-20)
### Added
 - Configurable, templated destination path for datasets when using Globus for transfer
 - (CONFIG) Transfer.Globus.DestinationTemplate contains a template for determining the path of the dataset on the destination side
### Changed
 - (CONFIG) Transfer.Globus.SourceCollection becomes Transfer.Globus.SourceCollectionID (no change in expected value)
 - (CONFIG) Transfer.Globus.DestinationCollection becomes Transfer.Globus.DestinationCollectionID (no change in expected value)
 - (CODE) Some refactoring, particularly there's now separate S3 and Globus add task functions
### Removed
 - (CONFIG) Transfer.Globus.DestinationPrefixPath is removed because DestinationTemplate replaces it in functionality

## [PR #126] (2025-04-29)
### Added
 - (Config) Transfer.ExtGlobus.CollectionRootPath is the path at which the Globus Collection is rooted. This value is used to convert absolute paths to Globus paths
### Changed
 - (Config) Webserver.Paths.CollectionLocation is now WebServer.Paths.CollectionLocations (plural), and its contents are now a map that maps collection location names (strings) to paths (strings).
 - (Code) the API now expects dataset paths to have their first path node to contain the collection location 'name'.  

A recommendation is to have the name of the collection's root folder to be the same as its "name" in the map. For example,  "example_collection" is  mapped to "/some/location/example_collection".