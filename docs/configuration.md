# OpenEM Data Network Ingestor

## Configuration File

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
    FrontendUrl: "http://scicat.example/ingestor"
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

- `FrontendUrl`: The url of the frontend should be put here, so that after login the backend can redirect there.
- `OAuth2.RedirectURL`: Host (localhost if running on desktop or host name when running as service) and port (same as Misc.Port) of the ingestor instance.
- `OIDC.IssuerURL`: replace `[KEYCLOAK_URL]` with URL of keycloak instance to be used
- `JwksURL`: replace `[KEYCLOAK_URL]` with URL of keycloak instance to be used
