# OpenEm Data Network Ingestor

This repository provides an ingestion app and service for dataset transfer and metadata registration in a catalog. It targets [Scicat](https://scicatproject.github.io) as dataset catalog.

Data can be transfered via [Globus](https://www.globus.org) or via S3 to a compatible endpoint.

There are two entrypoints, i.e. applications: a [desktop app](./cmd/openem-ingestor-app/) providing a minimal UI and a headless [service](./cmd/openem-ingestor-service/). Both provide a REST API in order to interact with each. Common functionality is extracted in the [core](./internal/) package.

## Core

The core package contains shared functionality between the desktop app and the service. It makes use of the [scicat-cli tools](https://github.com/paulscherrerinstitute/scicat-cli/tree/main) for interactions with Scicat. Two APIs are provided; a REST API for it interact with it as a service, and a Go API to interact with it within the same application.

## Desktop App

The desktop app is based on [wails.io](https://wails.io) which provides bindings between Go and typscript in order to write portable frontends in various web frameworks. Svelte was chosen in this case.

For development wails provides hot reload capabilities:

```bash
../Ingestor/desktop-app> wails dev
```

And a build command to build frontend and backend into a single executable:

```bash
../Ingestor/desktop-app> wails build
```

see [wails.io](https://wails.io) for details.

## Service

```bash
../Ingestor/desktop-app> go build
```

## Debugging

[launch.json](.vscode/launch.json) and [task.json](.vscode/tasks.json) are provided to define debug targets for VS Code.
