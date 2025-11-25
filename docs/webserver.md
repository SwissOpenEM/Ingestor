
# Webserver

## REST API

The API is defined using [OpenAPI specs](/api/openapi.yaml).

## Authentication and Access Control

### Summary

The server can be setup with an SSO provider using OAuth2 AuthZ protocol with the OIDC AuthN extension in order to verify the user identity and create a session for them.

Authentication can be disabled by setting `WebServer.Auth.Disable: true` in the configuration. This disables all checks, including directory-based access control, so it should only be used for local testing.

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
      - ingestor-read
      - ingestor-write
      - ingestor-admin
4. In the same realm, create a user:
    - username: `test`
    - password: `test`
    - email: `test@test.test`
    - Role mapping: assign `ingestor-read`, `ingestor-write`
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
        JwksURL: "http://[KEYCLOAK_URL]/realms/[REALM]/protocol/openid-conncerts"
        JwksSignatureMethods:
          - RS256
      RBAC:
        AdminRole: "ingestor-admin"
        CreateModifyTasksRole: "ingestor-write"
        ViewTasksRole: "ingestor-read"
    ```

6. To test if the auth works, go to [http://localhost:8888/login](http://localhost:8888/login) (if you haven't changed the defaults)
7. Login with the `test` account from step 4
8. Go into your browser's debugger and copy the `user` cookie created by the ingestor service
9. Use the following curl command: `curl --cookie "user=[USER_COOKIE]" -v "localhost:8888/transfer?page=1"`
10. If it is accepted, you have a working login session

### Configuration

The following section in the config file describes the necessary setup for authentication. Only OIDC is supported for SSO, and we don't provide any internal user login system.

```yaml
...
WebServer:
  Auth:
    ...
    FrontendUrl: "http://frontend.url" # optional value to set a redirect to a frontend.
    OAuth2:
      ClientID: "ingestor"
      RedirectURL: "http://localhost:8888/callback"
      Scopes:
        - email
    OIDC:
      IssuerURL: "http://oidc.provider/"
    JWT:
      UseJWKS: true
      JwksURL: "http://[OIDC_URL]/.../certs"
      JwksSignatureMethods:
        - RS256
    RBAC:
      AdminRole: "ingestor-admin"
      CreateModifyTasksRole: "ingestor-write"
      ViewTasksRole: "ingestor-read"
...
```

Please make sure the following fields are properly set:

- **WebServer.Auth.ClientID**: this is the client id of the ingestor. It should be added to the IdP that you want to use with the ingestor. This id shouldn't be shared with other ingestor instances. Look up your IdP's docs for adding a new client.
- **WebServer.Auth.OAuth2.RedirectURL**: The url at which the ingestor would be deployed. This should be known by you.
- **WebServer.Auth.OIDC.IssuerURL**: the url to the OIDC provider. It should conform to the Discovery spec. In case of Keycloak, it usually looks like `http://[KEYCLOAK_URL]/realms/[REALM_NAME]`.
- **WebServer.Auth.JWT.JwksURL**: It is the JwksURL of the OIDC provider. It is used to provide the client with the current set of public keys. It should have the same base url, but the rest of the path depends on the OIDC provider. In case of Keycloak, it should have the following format: `http://[KEYCLOAK_URL]/realms/[REALM_NAME]/protocol/openid-connect/certs`. If your provider does not support Jwks, then you can set the keys manually as follows:

```yaml
...
    JWT:
      UseJWKS: false
      Key: "[insert public key here]"
      KeySignMethod: "[set the key signature method here]"
...
```

- **WebServer.Auth.RBAC.[X]Role**: this is where you set your expected role names. It's a way to customize role names, but you can leave them as is. If facilities use shared OAuth2 client-id's (shouldn't be the case) then these roles should contain the name of each facility to make. You should also customize these if your IdP of choice can't separate what roles to map to users based on clientid. These roles specifically give permission to interact with the ingestor endpoints, and nothing else. Accessing datasets is determined by the `AccessGroups` of the user on SciCat.

{: .box-note}
If you're using the supplied example scicatlive config for testing, the roles are named `FAC_ingestor_[function]` where `[function]` can be "admin", "write" or "read".

{: .box-note}
**Note:** If your IdP isn't keycloak you have to make sure that the roles are mapped to OAuth2 claims in the same way as Keycloak: `[access_token_jwt].resource_access[(client_id)].roles`

## Paths

```yaml
...
WebServer:
  Paths:
    CollectionLocations:
      location1: "/some/path/location1"
      Projects: "/some/other/path/location2"
    ExtractorOutputLocation: "(optional)/location/to/output/temp/files"
...
```

- It's important configure `CollectionLocation` as that is where the ingestor will look for to find datasets.
- The ExtractorOutputLocation sets a custom path for the temporary extractor files. Normally they're outputted to /tmp.
- Due to the way the config library works, all location keys will be lowercased.

## Configuration

```yaml
WebServer:
  Auth:
    Disable: false
    Frontend:
      Origin: ${SCICAT_FRONTEND_URL}
      RedirectPath: "/ingestor"
    SessionDuration: 28800
    OAuth2:
      ClientID: "${KEYCLOAK_CLIENT_ID}"
      RedirectURL: "${INGESTOR_DOMAIN}/callback"
      Scopes:
        - email
    OIDC:
      IssuerURL: "${KEYCLOAK_URL}/realms/${KEYCLOAK_REALM}"
    JWT:
      UseJWKS: true
      JwksURL: "${KEYCLOAK_URL}/realms/${KEYCLOAK_REALM}/protocol/openid-connect/certs"
      JwksSignatureMethods:
        - RS256
    RBAC:
      AdminRole: "ingestor-admin"
      CreateModifyTasksRole: "ingestor-write"
      ViewTasksRole: "ingestor-read"
  Paths:
    CollectionLocations:
      ${HOST_COLLECTION_NAME}: ${HOST_COLLECTION_PATH}
  MetadataExtJobs:
    ConcurrencyLimit: 4
    QueueSize: 200
  Other:
    BackendAddress: ${INGESTOR_DOMAIN}
    Port: 8080
    SecureCookies: true
    LogLevel: Info 
    DisableServiceAccountCheck: true
```

### Directory access control

By default, users which are logged in to scicat and the ingestor will be able to browse all files within collection locations (set in the `WebServer.Paths.CollectionLocations` config section). Access to individual directories can be controlled by adding a file `.ingestor-access.yaml`.

Example `.ingestor-access.yaml`:

```yaml
# Only users of these groups can browse this directory
# Defaults to all users
AllowedGroups:
  - PSI/gatan-users
# Users of these groups will never be allowed, even if they are in an AllowedGroup
# Defaults to []
BlockedGroups:
  - PSI/trainees
# Hint to the user that subdirectories are valid datasets for ingestion
# Otherwise, the hint is set by whether a directory contains only files
HasDatasetFolders: true
```
