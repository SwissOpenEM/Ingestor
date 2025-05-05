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

## [PR #127] (2025-05-05)
### Added
 - (CONFIG) Transfer.StorageLocation sets an ID string in the dataset lifecycle that specifies to which facility we're transmitting the dataset's data