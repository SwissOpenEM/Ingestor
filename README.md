# OpenEm Data Network Ingestor

This repository provides an ingestion app and service for dataset transfer and metadata registration in a catalog. It targets [Scicat](https://scicatproject.github.io) as dataset catalog.

Data can be transfered via [Globus](https://www.globus.org) or via S3 to a compatible endpoint.

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

## Configuration

Both the desktop app and the service will use a configuration file named  `openem-ingestor-config.yaml` expected to be located next to the executable or in `os.UserConfigDir()/openem-ingestor` (with [os.UserConfigDir from Golang](https://pkg.go.dev/os#UserConfigDir)) where the first takes precedence.

```yaml
Scicat:
  Host: http://scicat:8080/api/v3
  AccessToken: "token"
Transfer:
  Method: S3
  S3:
    Endpoint: s3:9000
    Bucket: landingzone
    Checksum: true
  Globus:
    Endpoint: globus.psi.ch
Misc:
  ConcurrencyLimit: 2
  Port: 8888
```

## Debugging

[launch.json](.vscode/launch.json) and [task.json](.vscode/tasks.json) are provided to define debug targets for VS Code.
