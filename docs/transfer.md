# Transfer

The ingestor service provides two ways to transfer, Globus and S3. Whereas Globus requires [Globus Connect Server](https://docs.globus.org/globus-connect-server/v5/) to be installed, S3 requires and S3 storage endpoint as well an [Archiver-API-Service](https://github.com/SwissOpenEM/ScopeMArchiver) to be deployed.

## Globus using PSI Transfer Request Service (recommend)

This methods uses and external services to request the transfers via Globus, the [Globus-Proxy](https://github.com/SwissOpenEM/scicat-globus-proxy). The is the recommended way of using Globus.

```yaml
Transfer:
  Method: ExtGlobus
  ExtGlobus:
    TransferServiceUrl: "https://url.at.psi/globus/service"
    SrcFacility: "EXAMPLE-FACILITY-1" # "FAC-1" if you're using the default scicatlive setup
    DstFacility: "EXAMPLE-FACILITY-2" # "FAC-2" if you're using the default scicatlive setup
    CollectionRootPath: "/some/path" # the path at which the Source Globus Collection is mounted (eg. '/home')
```

> **Disable service account check**: using this mode, the `webserver.other.DisableServiceAccountCheck` should be set to `true`, as there's no need for any service account in the Ingestor in this mode.

## S3

Transferring via S3 requires an [Archiver-API-Service](https://github.com/SwissOpenEM/ScopeMArchiver) to be running.

```yaml
Transfer: 
  Method: S3
  S3:
    Endpoint: https://s3.example.ethz.ch/archiver/api/v1
    TokenUrl: https://kc.psi.ch/keycloak/realms/facility/protocol/openid-connect/token
    ClientID: archiver-service-api
    ChunkSizeMB: 128
    ConcurrentFiles: 4
    PoolSize: 12
```

**Endpoint**: Endpoint of the archiver api service
**TokenUrl**: Endpoint for OIDC token
**ChunkSizeMB**: Size of chunks when doing mulitpart uploads
**CurrentFiles**: Number of files uploaded concurrently
**PoolSize**: Number of Goroutines running concurrently for all the uploads

The last 3 parameters have crucial impact on performance and may be tuned to the specific use case.

Please refer to [ScopeMArchiver](https://github.com/SwissOpenEM/ScopeMArchiver) for more information.

## Direct Globus (deprecated)

This method interacts directly with Globus and therefore requires explicit configuration:

```yaml
...
Transfer:
  Method: Globus
  Globus:
    ClientID: "globus-auth-client-id"
    ClientSecret: "globus-auth-client-secret[optional]"
    RedirectURL: "[insert ingestor frontend url]"
    Scopes:
      - scope1
      - scope2
      ...
    SourceCollectionID: "uuid-of-source-collection"
    CollectionRootPath: "/insert/path/here"
    DestinationCollectionID: "uuid-of-destination-collection"
    DestinationTemplate: "/nacsa/{{ .Username }}/{{ replace .Pid \".\" \"_\" }}/{{ .DatasetFolder }}"
...
```

**Transfer.Globus.ClientID**: this should be set to the same client-id as the one you'll use in the next paragraph. You need to create a new client on `app.globus.org`, please refer to the [Globus documentation]([documentation/admin/installation/globus.md](https://www.openem.ch/documentation/admin/installation/globus)) for more information.

**Scopes**: These will include scopes for accessing the Globus Connect Server endpoints you want to interact with in the name of the user. Usually, you're only required to specify the following scope for each endpoint: `"urn:globus:auth:scope:transfer.api.globus.org:all[*https://auth.globus.org/scopes/[ENDPOINT ID HERE]/data_access]"` where you replace `[ENDPOINT ID HERE]` with the endpoint's UUID.

> **Note:** The source and destination endpoint scopes are only intended for Globus Connect Server endpoints. For Globus Connect Personal (GCP), just skip specifying the scope made from its `collection-id`. You have to make sure that the GCP collection is owned by the token's user.

**Service account**: using this mode, the `webserver.other.DisableServiceAccountCheck` should be set to `false`, and a service account must be set using the `INGESTOR_SERVICE_USER_NAME` and `INGESTOR_SERVICE_USER_PASS` environment variables. These are the credentials for an internal SciCat user, which has the right to update any dataset. It is needed in order to safely mark any dataset as archivable in this mode.
