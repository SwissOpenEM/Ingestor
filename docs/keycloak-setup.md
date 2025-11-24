# Keycloak Development Setup

## Adding the Ingestor to Keycloak

> **Note:** This section assumes you're running the Ingestor locally.
Replace all instances of `http://localhost:8888/` if you've deployed it in some other way.

### Keycloak Setup

1. Setup keycloak, preferably with Docker
2. [OPTIONAL] Add another realm where you'll have your ingestor client added.

  <div align="center">
    <img src="img/keycloak/img0.png" alt="Centered Image" width="20%">
  </div>

3. Add a new client with the following parameters

  <div align="center">
    <img src="img/keycloak/img1.png" alt="Centered Image" width="20%">
    <img src="img/keycloak/img2.png" alt="Centered Image" width="20%">
    <img src="img/keycloak/img3.png" alt="Centered Image" width="20%">
  </div>

4. Edit your client and add client-specific roles that match the ones from your Ingestor config

  <div align="center">
    <img src="img/keycloak/img5.png" alt="Centered Image" width="20%">
  </div>

5. Under the client's "Client Scopes" tab, click on `ingestor-dedicated`
6. `Add mapper` button -> "By configuration" -> Group Membership
7. The `token claim name` should be "accessGroups" and `Full group path` should be *turned off*

>**Note:** In most cases you will be using some external source of users in Keycloak, in which case, you need to map some claim of the incoming user to the roles that were setup in Step 4. This is not covered in this Install guide as it is highly specific to your own setup. If by any chance you're setting up users directly in Keycloak, you can assign them the roles directly within the Keycloak admin menu.

### Testing with authentication enabled locally (Keycloak dev setup)

1. Add a new test user. Don't forget to set a password.

<div align="center">
  <img src="img/keycloak/img7.png" alt="Centered Image" width="20%">
  <img src="img/keycloak/img8.png" alt="Centered Image" width="20%">
  <img src="img/keycloak/img9.png" alt="Centered Image" width="20%">
</div>

2. Assign the read and write roles of the ingestor to this user.

  <div align="center">
    <img src="img/keycloak/img10.png" alt="Centered Image" width="20%">
    <img src="img/keycloak/img11.png" alt="Centered Image" width="20%">
  </div>
3. Go to [http://localhost:8888/login](http://localhost:8888/login)
4. This will open up the keycloak login page. Use your test user for logging in.
  <div align="center">
    <img src="img/keycloak/img12.png" alt="Centered Image" width="20%">
  </div>
5. If everything went well, you should be redirected to `RedirectURL`, and you should
  see a "user" cookie associated to the `localhost` domain in your browser's debugger.
  If you also have a valid `FrontendUrl` your browser will get redirected to your
  Ingestor frontend, where you should be able to interact with the ingestor backend
  using the cookie.
  <div align="center">
    <img src="img/keycloak/img13.png" alt="Centered Image" width="20%">
  </div>
6. [OPTIONAL] To test the ingestor's auth directly, copy the cookie value from the browser, then you can use the following curl command:

```bash
curl -v --cookie "user=[INSERT COOKIE VALUE HERE]" "localhost:8888/transfer?page=1"
```

If the auth is successful, you should get an empty list as a reply.