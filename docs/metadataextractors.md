# Metadata Extractors

Metadata extractors are invoked as external binaries that need to be installed and available at a configurable location. The extractors can be invoked by a configurable commandline and are expected to produce a metadata as a valid json file.

## Installing Metadata Extractors

Metadata extractors are external binaries called by the ingestor with a command line template.

Example:

```yaml
CommandLineTemplate: "-i '{{.SourceFolder}}' -o '{{.OutputFile}}'"
```

`{{.SourceFolder}}` and `{{.OutputFile}}` are values provided by the ingestor to designate the folder with the dataset and the output file (.json), respectively.

> **Note**: The quotes are required to handle whitespaces in paths correctly.

Additional parameters can be either added directly to the command line template

```yaml
CommandLineTemplate: "-i '{{.SourceFolder}}' -o '{{.OutputFile}}' -p SomeValue"
```

or as a list in yaml

```yaml
CommandLineTemplate: "-i '{{.SourceFolder}}' -o '{{.OutputFile}}' {{.AdditionalParameters}}"`
AdditionalParameters:
  - Param1=SomeValue1
  - Param2=SomeValue2
```

### 1. Manual Installation

When installing extractors manually, the executable is expected to be in the following location:
`{{.InstallationPath}}/{{.GithubOrg}}/{{.GithubProject}}/{{.Version}}/{{.Executable}}`

### 2. Automatic Download from Github

Alternatively, the ingestor can download metadata extractors from github releases if `DownloadMissingExtractors` is set to `true`. It will download and unpack the respective package into the correct folder, as well as verify the checksum of the downloaded package.
The packages needs to contain the architecture designator in their name, e.g. `LS_Metadata_reader_Linux_x86_64.tar.gz`

### Metadata Schemas

Methods in the metadata extractor depend on schemas which are downloaded from a Url during startup of the ingestor. The schemas will be downloaded from the given Url and presented in the UI with name given in `Name`.

Example config:

```yaml
    Methods:
      - Name: Single Particle
        Schema: oscem_schemas_spa.schema.json
        Url: https://raw.githubusercontent.com/osc-em/OSCEM_Schemas/refs/heads/main/project/spa/jsonschema/oscem_schemas_spa.schema.json

```

### Configuration Example

The following examples shows two extractors with multiple schemas.

```yaml
MetadataExtractors:
  InstallationPath: ./extractors/
  SchemasLocation: ./schemas/
  DownloadSchemas: true
  DownloadMissingExtractors: true
  Timeout: 20m
  Extractors:
    - Name: LS
      GithubOrg: SwissOpenEM
      GithubProject: LS_Metadata_reader
      Version: v2.0.1
      Executable: LS_Metadata_reader
      Checksum: 83ae1b2d469cec10fdcc3ad7cb6c824a389e501e7142417792f2ac836ab174c3
      ChecksumAlg: sha256
      CommandLineTemplate: "-i '{{.SourceFolder}}' -o '{{.OutputFile}}' --cs ${INSTRUMENT_CS_VALUE} --folder_filter=^.*(?i)(metadata).*$"
      Methods:
        - Name: Single Particle
          Schema: oscem_schemas_spa.schema.json
          Url: https://osc-em.github.io/oscem-schemas/artifacts/latest/spa/jsonschema/oscem_schemas_spa.schema.json
        - Name: Cellular Tomography
          Schema: oscem_cellular_tomo.json
          Url: https://osc-em.github.io/oscem-schemas/artifacts/latest/cellular_tomo/jsonschema/oscem_schemas_cellular_tomo.schema.json
        - Name: Tomography
          Schema: oscem_tomo.json
          Url: https://osc-em.github.io/oscem-schemas/artifacts/latest/subtomo/jsonschema/oscem_schemas_subtomo.schema.json
        - Name: Environmental Tomography
          Schema: oscem_env_tomo.json
          Url: https://osc-em.github.io/oscem-schemas/artifacts/latest/env_tomo/jsonschema/oscem_schemas_env_tomo.schema.json

    - Name: MS
      GithubOrg: SwissOpenEM
      GithubProject: MS_Metadata_reader
      Version: v1.0.3
      Executable: MS_Metadata_reader
      Checksum: 56925ded88d6719d42eaafe66f98fce43a67f0132d6488ece8bcbcfd5f0704a3
      ChecksumAlg: sha256
      CommandLineTemplate: "'{{.SourceFolder}}' '{{.OutputFile}}'"
      Methods:
        - Name: Material Science
          Schema: oscem_general.json
          Url: https://osc-em.github.io/oscem-schemas/artifacts/latest/general/jsonschema/oscem_schemas_general.schema.json
```

- **InstallationPath** determines where the extractors should be downloaded/installed.
- **SchemasLocation** determines where the schemas for extractors are downloaded to.
- **DownloadSchemas** sets whether to download the schemas
- **DownloadMissingExtractors** sets whether to download extractors automatically from github
- **Timeout** sets the maximal time any extractor should run before timing out
- **Extractors** is the list of extractors.
  - if using github for downloading, the following link is used `https://github.com/[GithubOrg]/[GithubProject].git` to look for matching releases
  - **Version`** is the *release tag* that will be attempted to be used.
  - **Executable** is the file that will be executed. Might have different names on different platforms.
  - **Checksum** is used to verify the integrity of the executable
  - **ChecksumAlg** is to define the algorithm used for the checksum (only sha256 is used)
  - **CommandLineTemplate** is the command template to use with the executable, it appends a formatted list of paramters.
  - **Methods** is where you can define a list of methods that can be used with a particular extractor.
    - **Name** is the name of the method
    - **Schema** is the metadata schema to use for this method (must exist in **SchemasLocation**)
    - **Url** is the url for the schema, it will be used when the schema is not found locally to download it.

### Metadata Extractor Jobs

This section is for configuring the metadata extractor job system. It is a system to process extraction requests in parallel and in order of requests.

```yaml
WebServer:
  MetadataExtJobs:
    ConcurrencyLimit: 4
    QueueSize: 200
```

Where the **ConcurrencyLimit** is the max. number of extractions to be executed in parallel, and **QueueSize** is the max queue size which has FIFO order.
If there are more pending requests than **QueueSize** then those requests will be processed randomly.
