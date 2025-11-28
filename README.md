# OpenEM Network Ingestor

[![Build and Test](https://github.com/SwissOpenEM/Ingestor/actions/workflows/ci.yaml/badge.svg)](https://github.com/SwissOpenEM/Ingestor/actions/workflows/ci.yaml)

> Please refer to [https://www.openem.ch/](https://www.openem.ch/) for an overview.

This repository provides a data ingestion service for dataset transfer and metadata registration in a catalog. It targets [Scicat](https://scicatproject.github.io) as dataset catalog.

Data can be transferred via [Globus](https://www.globus.org) or to an S3 compatible endpoint.

The main entrypoint is a headless [service](./cmd/openem-ingestor-service/) that provides a REST API.

## Building the Service

### Generating the REST API

Based on the OpenApi specs in [openapi.yaml](./api/openapi.yaml), the REST API for the server implementation ([Gin](https://gin-gonic.com) can be built:

```bash
/Ingestor$ go generate ./...
```

this will update [api.gen.go](./internal/webserver/api.gen.go).

```bash
/Ingestor$ go build ./cmd/ingestor-web-service
```

## Debugging

[launch.json](.vscode/launch.json) and [task.json](.vscode/tasks.json) are provided to define debug targets for VS Code. Running the `Debug Service` task will start the service on the configured port (default: 8888) and a Swagger UI documentation page can be accessed at

<http://localhost:8888/docs/index.html>

## Configuration

Configuration options are described in [configs/ReadMe.md](configs/ReadMe.md)

Both the desktop app and the service will use a configuration file named  `openem-ingestor-config.yaml` expected to be located next to the executable or in `os.UserConfigDir()/openem-ingestor` where the first takes precedence. As documented in the [Go documentation](https://pkg.go.dev/os#UserConfigDir)), the following config locations are considered:

- Unix: `$XDG_CONFIG_HOME/openem-ingestor/openem-ingestor-config.yaml` if non-empty, else `$HOME/.config/openem-ingestor/openem-ingestor-config.yaml`
- MacOS: `$HOME/Library/Application Support/openem-ingestor/openem-ingestor-config.yaml`
- Windows: `%AppData%\openem-ingestor\openem-ingestor-config.yaml`

See [configs/openem-ingestor-config.yaml](configs/openem-ingestor-config.yaml) for an example configuration.

## Core

The core package contains main functionality. It makes use of the [scicat-cli tools](https://github.com/paulscherrerinstitute/scicat-cli/tree/main) for interactions with Scicat. Two APIs are provided; a REST API for it interact with it as a service, and a Go API to interact with it within the same application.

## Further Documentation

- General Configuration: [docs/configuration.md](docs/configuration.md)
- Metadata Extractors: [docs/metadataextractors.md](docs/metadataextractors.md)
- Transfer: [docs/transfer.md](docs/transfer.md)
- WebServer: [docs/authentication.md](docs/webserver.md)
- Keycloak Setup for development: [docs/keycloak-setup.md](docs/keycloak-setup.md)

## Deployment

For deployment instruction and example setup see [openem-deployment](https://github.com/SwissOpenEM/openem-deployment) repository.
