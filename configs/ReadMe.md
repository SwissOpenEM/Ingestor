# OpenEM Data Network Ingestor

## Configuration

The configuration file `openem-ingestor.config.yaml` can be put into two locations:

1. Next to the executable
2. Into `$USERCONFIGDIR/openem-ingestor` where `$USERCONFIGDIR` is resolved like this:

```
On Unix systems, it returns $XDG_CONFIG_HOME as specified by
https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if
non-empty, else $HOME/.config.
On Darwin, it returns $HOME/Library/Application Support.
On Windows, it returns %AppData%.
``` 

see <https://pkg.go.dev/os#UserConfigDir/>

>**Note**: the first one takes precedence.

### Installing Metadata Extractors

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

1. Manual Installation

When installing extractors manually, the executable is expected to be in the following location:

`{{.InstallationPath}}/{{.GithubOrg}}/{{.GithubProject}}/{{.Version}}/{{.Executable}}`

2. Download from Github
Alternatively, the ingestor can download metadata extractors from github releases if `DownloadMissingExtractors` is set to `true`. It will download and unpack the respective package into the correct folder, as well as verify the checksum of the downloaded package.
The packages needs to contain the architecture designator in their name, e.g. `LS_Metadata_reader_Linux_x86_64.tar.gz`