package webserver

import "github.com/golang-jwt/jwt/v5"

type rolesList struct {
	Roles []string `json:"roles,omitempty"`
}

type idTokenClaims struct {
	// the `azp` (Authorized Party) claim. See https://openid.net/specs/openid-connect-core-1_0.html#IDToken
	AuthorizedParty string `json:"azp,omitempty"`
}

type keycloakClaims struct {
	RealmAccess    rolesList            `json:"realm_access,omitempty"`
	ResourceAccess map[string]rolesList `json:"resource_access,omitempty"`
	idTokenClaims
	jwt.RegisteredClaims
}

func (c *idTokenClaims) GetAuthorizedParty() string {
	return c.AuthorizedParty
}

func (c *keycloakClaims) GetRealmRoles() []string {
	return c.RealmAccess.Roles
}

func (c *keycloakClaims) GetResourceRolesByKey(key string) []string {
	if val, ok := c.ResourceAccess[key]; ok {
		return val.Roles
	} else {
		return []string{}
	}
}

func (c *keycloakClaims) GetAllResourceRoles() map[string][]string {
	returnMap := make(map[string][]string)
	for k, v := range c.ResourceAccess {
		returnMap[k] = v.Roles
	}
	return returnMap
}
