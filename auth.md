# Authentication Flows

Requests related to authentication are dotted lines, while other API calls are solid
lines.

## Option A: Validate Scicat identity

```mermaid
sequenceDiagram
  autonumber
  participant B as Browser
  participant S as Scicat Backend
  participant I as Ingestor Service
  participant G as Storage
  participant A as ETHZ Archiver Service
  participant K as Keycloak


  B --) S: Login
  S --) B: Scicat-JWT🔑
  B --) I: Login with Scicat-JWT
  I -->> S: Validate Scicat-JWT (/api/v3/users/{id}/userIdentity)
  S -->> I: Return roles and groups
  I --) B: Ingestor-JWT🔑
  B -) I: POST /ingest (Ingestor-JWT) passing Scicat-JWT (and Globus access token)
  I -) S: POST /dataset (Scicat-JWT)

  alt ETHZ
    I --) A: POST /token (Scicat-JWT) with pid
    A -->> S: Validate Scicat-JWT (/api/v3/users/{id}/userIdentity)
    S -->> A: Return roles and groups

    A --) K: POST request transfer token (service account)
    K --) A: Token for given dataset id: transfer-JWT
    A --) I: transfer-JWT🔑

    I ->> A: POST /presigned (transfer-JWT)
    A ->> I: presigned URLs

    I -) G: Transfer data (presigned URLs)
    I -) A: Complete upload (transfer-JWT)
    A -) S: PATCH /dataset (service account)

  else Globus
    I -) G: Transfer (Globus access token)
    I -) S: PATCH /dataset (Service User)
  end

```


- Both the ingestor and archiver accept scicat tokens during logon.
- The ingestor/archiver validates the scicat token using a `/userinfo` endpoint and
  check the returned payload for authorization claims.
- Requires a scicat service user for the ingestor for the dataset update for globus.
  ETHZ can avoid this by re-using the archiver service user (via an api)

### Changes needed

- (scicat backend) Add authorization claims to `/userinfo`


## Option B

```mermaid
sequenceDiagram
  autonumber
  participant B as Browser
  participant S as Scicat Backend
  participant I as Ingestor Service
  participant G as Storage (Globus)
  participant A as ETHZ Archiver Service
  participant K as Keycloak

  B --) I: Login with keycloak
  I -->> I: store refresh-token
  I --) B: Ingestor-JWT🔑
  B -) I: POST /ingest (Ingestor-JWT) (with Globus access token)

  I ->> S: Login (Ingestor-JWT)
  S ->> I: Scicat-JWT🔑
  I -) S: POST /dataset (Scicat-JWT)

  alt ETHZ
    I --) A: POST /token (Ingestor-JWT)
    A -->> A: verify Ingestor-JWT

    A --) K: POST request transfer token (service account)
    K --) A: Token for given dataset id: transfer-JWT
    A --) I: transfer-JWT🔑

    I ->> A: POST /presigned (transfer-JWT)
    A ->> I: presigned URLs

    I -) G: Transfer data (presigned URLs)

  else Globus
    I -) G: Transfer (Globus access token)
  end


    I --) K: Renew token (refresh-token)
    K --) I: Ingestor-JWT-new🔑
    I -->> S: Login (Ingestor-JWT-new)
    S -->> I: Scicat-JWT-new🔑

    I -) S: PATCH /dataset (Scicat-JWT-new)

```

- The user doesn't pass the Scicat-JWT to the ingestor at any time. Instead, the
  ingestor can directly exchange the Ingestor-JWT (which is issued by keycloak and
  contains all needed claims) for a scicat token.
- Ingestor tokens are issued with a refresh_token, allowing them to be renewed after the
  data transfer is complete

## Changes

- Accept Ingestor-JWT as a valid login method. This may require token exchange, since
  scicat and the ingestor have different clientIds
