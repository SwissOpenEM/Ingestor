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

## [PR #127] (2025-05-05)
### Added
 - (CONFIG) Transfer.StorageLocation sets an ID string in the dataset lifecycle that specifies to which facility we're transmitting the dataset's data

## [PR #135] (2025-05-14)
### Changed
 - (Config) Transfer.Globus.SourcePrefixPath was changed to Transfer.Globus.CollectionRootPath due to the change regarding multiple collection locations. 

This changed value **does not** function in the same way: instead of adding a prefix path to the path given as SourceFolder of the dataset, it is now applied after the collection location resolution, making the absolute path relative to the root of the Globus collection. 

Basically, if `location1` is a collection location that points to `/datasets/locations/location1`, and the user gives "/location1/dataset1" as a dataset `SourceFolder`, then the ingestor will first transform it to `/datasets/locations/location1/dataset1`. 

Then, if the `CollectionRootPath` is set to `/datasets/locations`, then that means the Globus collection's root path is mounted there in the filesystem, and the Ingestor will transform the path in a way that removes the upper levels of the path, resulting in `/location1/dataset1` as the source path when requesting a transfer from Globus.