# OpenEm Data Network Ingestor

This repository provides an ingestion app and service for dataset transfer and metadata registration in a catalog. It targets [Scicat](https://scicatproject.github.io) as dataset catalog.

Data can be transferred via [Globus](https://www.globus.org) or via S3 to a compatible endpoint.

There are two entrypoints, i.e. applications: a [desktop app](./cmd/openem-ingestor-app/) providing a minimal UI and a headless [service](./cmd/openem-ingestor-service/). Both provide a REST API in order to interact with each. Common functionality is extracted in the [core](./internal/) package.

## Core

The core package contains shared functionality between the desktop app and the service. It makes use of the [scicat-cli tools](https://github.com/paulscherrerinstitute/scicat-cli/tree/main) for interactions with Scicat. Two APIs are provided; a REST API for it interact with it as a service, and a Go API to interact with it within the same application.

### Generating the REST API

Based on the OpenApi specs in [openapi.yaml](./api/openapi.yaml), the REST API for the server implementation ([Gin](https://gin-gonic.com) can be built:

```bash
../Ingestor/internal/webserver> go generate
```

this will update [api.gen.go](./internal/webserver/api.gen.go).

## Building the Desktop App

The desktop app is based on [wails.io](https://wails.io) which provides bindings between Go and typscript in order to write portable frontends in various web frameworks. Svelte was chosen in this case.

For development wails provides hot reload capabilities:

```bash
../Ingestor/desktop-app> wails dev
```

And a build command to build frontend and backend into a single executable:

```bash
../Ingestor/cmd/openem-ingestor-app> wails build
```

see [wails.io](https://wails.io) for details.

## Building the Service

```bash
../Ingestor/cmd/openem-ingestor-service> go build
```

## Running the container using Docker Compose

The ingestor service can be run in a docker container and be deployed using docker compose.

```bash
docker compose up -d
```

The following environment variables need to be set (e.g. using an `.env` file):

| Variable               | Description                                                                                                  | Example Value                          |
|------------------------|--------------------------------------------------------------------------------------------------------------|----------------------------------------|
| `NFS_SERVER_ADDRESS`   | Address of NFS fileserver where dataset are stored                                                           | `192.168.1.1`, `nfs-server.facilty.ch` |
| `UID`                  | User id as which the container runs. This might be relevant to access certain datatasets if run as non-root  | `1000`                                 |
| `GID`                  | Group id as which the container runs. This might be relevant to access certain datatasets if run as non-root | `1000`                                 |
| `HOST_COLLECTION_PATH` | Path to folder on the host system where dataset are stored. Can also be a mounted folder.                    | `/mnt/datasets`                        |
| `KEYCLOAK_URL`         | Url to keycloak instance used with Scicat                                                                    | `https://kc.psi.ch`                    |
| `SCICAT_FRONTEND_URL`  | Url to Scicat brontend                                                                                       | `https://discovery.psi.ch`             |
| `SCICAT_BACKEND_URL`   | Url to Scicat backend                                                                                        | `https://dacat.psi.ch`                 |

The ingestor configuration is set directly as a `config` top-level element in docker-compose.yaml.

## Configuration

Configuration options are described in [configs/ReadMe.md](configs/ReadMe.md)

Both the desktop app and the service will use a configuration file named  `openem-ingestor-config.yaml` expected to be located next to the executable or in `os.UserConfigDir()/openem-ingestor` where the first takes precedence. As documented [here](https://pkg.go.dev/os#UserConfigDir)), the following config locations are considered:
- Unix: `$XDG_CONFIG_HOME/openem-ingestor/openem-ingestor-config.yaml` if non-empty, else `$HOME/.config/openem-ingestor/openem-ingestor-config.yaml`
- MacOS: `$HOME/Library/Application Support/openem-ingestor/openem-ingestor-config.yaml`
- Windows: `%AppData%\openem-ingestor\openem-ingestor-config.yaml`

See [configs/openem-ingestor-config.yaml](configs/openem-ingestor-config.yaml) for an example configuration.

## Authentication and Access Control

### Summary

The server can be setup with an SSO provider using OAuth2 AuthZ protocol with the OIDC AuthN extension in order to verify the user identity and create a session for them. 

### Technical details

 - The server uses the provider's token to estabilish its own user session
 - It does not directly accept bearer tokens estabilished by the SSO provider
 - It creates an HttpOnly cookie based user session using the claims provider by the IdP (SSO Provider)
 - This basically means that the server can't function as a "Resource server", you need a specific session with it
 - Currently 3 basic roles exist: `Admin`, `CreateModifyTasks` and `ViewTasks`
 - The roles' names can be defined in the config (eg. to add the facility name in the role name)
 - These roles should be associated with the server's ClientId
 - The roles must be served under the following claim in the `access_token`: `resource_access/[ClientId]/roles`, where `roles` is a list of strings
 - Keycloak serves client-specific roles assigned to the user under the claim mentioned above by default

### A typical Keycloak setup for development

1. Create a keycloak instance (docker is recommended)
2. Create a new realm (recommended, but you can use the master realm too)
3. In that realm, create a client:
    - ClientID: `ingestor`
    - Root URL: `http://localhost:8888`
    - Home URL: `/`
    - Valid redirect URIs: `*`
    - Valid post logout redirect URIs: `*`
    - Add the following roles:
      - FACILITY-ingestor-read
      - FACILITY-ingestor-write
      - FACILITY-ingestor-admin
4. In the same realm, create a user:
    - username: `test`
    - password: `test`
    - email: `test@test.test`
    - Role mapping: assign `FACILITY-ingestor-read`, `FACILITY-ingestor-write`
5. Make sure you have the following section in your ingestor config file:

```yaml
WebServerAuth:
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

6. To test if the auth works, go to [http://localhost:8888/login](http://localhost:8888/login) (if you haven't changed the defaults)
7. Login with the `test` account from step 4
8. Go into your browser's debugger and copy the `user` cookie created by the ingestor service
9. Use the following curl command: `curl --cookie "user=[USER_COOKIE]" -v "localhost:8888/transfer?page=1"`
10. If it is accepted, you have a working login session

Todo: alternate setup with an ingestor frontend

## Metadata Extractors

Metadata extractors are invoked as external binaries that need to be installed and available at a configurable location. The extractors can be invoked by a configurable commandline and are expected to produce a metadata as a valid json file. See [configs/ReadMe.md](configs/ReadMe.md#installing-metadata-extractors) for details

## Debugging

[launch.json](.vscode/launch.json) and [task.json](.vscode/tasks.json) are provided to define debug targets for VS Code.
