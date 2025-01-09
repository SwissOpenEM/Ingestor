# OpenEM Data Network Ingestor

## Configuration

The configuration file `openem-ingestor.config.yaml` can be put into two locations:

1. Next to the executable (taking precedence)
2. Into `$USERCONFIGDIR/openem-ingestor` where `$USERCONFIGDIR` is resolved like this:

   - Unix: `$XDG_CONFIG_HOME/openem-ingestor/openem-ingestor-config.yaml` if non-empty, else `$HOME/.config/openem-ingestor/openem-ingestor-config.yaml`
   - MacOS: `$HOME/Library/Application Support/openem-ingestor/openem-ingestor-config.yaml`
   - Windows: `%AppData%\openem-ingestor\openem-ingestor-config.yaml`

  see <https://pkg.go.dev/os#UserConfigDir/> for details.

### Authentication

The following section in the config file describes the necessary setup for authentication.

```yaml
WebServer:
  Auth:
    Disable: false
    SessionDuration: 28800
    OAuth2:
      ClientID: "ingestor"
      RedirectURL: "http://localhost:8888/callback"
      Scopes:
        - email
    OIDC:
      IssuerURL: "http://[KEYCLOAK_URL]/realms/facility"
    JWT:
      UseJWKS: true
      JwksURL: "http://[KEYCLOAK_URL]/realms/facility/protocol/openid-connect/certs"
      JwksSignatureMethods:
        - RS256
    RBAC:
      AdminRole: "FACILITY-ingestor-admin"
      CreateModifyTasksRole: "FACILITY-ingestor-write"
      ViewTasksRole: "FACILITY-ingestor-read"
```

The necessary fields to adapt are

- `OAuth2.RedirectURL`: Host (localhost if running on desktop or host name when running as service) and port (same as Misc.Port) of the ingestor instance.
- `OIDC.IssuerURL`: replace `[KEYCLOAK_URL]` with URL of keycloak instance to be used
- `JwksURL`: replace `[KEYCLOAK_URL]` with URL of keycloak instance to be used


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

#### 1. Manual Installation

When installing extractors manually, the executable is expected to be in the following location:
`{{.InstallationPath}}/{{.GithubOrg}}/{{.GithubProject}}/{{.Version}}/{{.Executable}}`

#### 2. Download from Github
  
Alternatively, the ingestor can download metadata extractors from github releases if `DownloadMissingExtractors` is set to `true`. It will download and unpack the respective package into the correct folder, as well as verify the checksum of the downloaded package.
The packages needs to contain the architecture designator in their name, e.g. `LS_Metadata_reader_Linux_x86_64.tar.gz`