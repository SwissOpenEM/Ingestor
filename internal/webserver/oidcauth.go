package webserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
)

type rolesList struct {
	Roles []string `json:"roles,omitempty"`
}

type keycloakClaims struct {
	RealmAccess    rolesList            `json:"realm_access,omitempty"`
	ResourceAccess map[string]rolesList `json:"resource_access,omitempty"`
	jwt.RegisteredClaims
}

func (c *keycloakClaims) GetRealmRoles() []string {
	return c.RealmAccess.Roles
}

func (c *keycloakClaims) GetResourceRolesByClient(clientName string) []string {
	return c.ResourceAccess[clientName].Roles
}

func initKeyfunc(jwtConf core.JWTConf) (jwt.Keyfunc, error) {
	if jwtConf.UseJWKS {
		jwks, err := keyfunc.NewDefault([]string{jwtConf.JwksURL})
		if err != nil {
			return nil, err
		}
		return jwks.Keyfunc, nil
	} else {
		return func(token *jwt.Token) (interface{}, error) {
			// validate signing algorithm
			if token.Header["alg"] != jwtConf.KeySignMethod {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// return public key if the algorithm is what is expected
			switch jwtConf.KeySignMethod {
			case "HS256", "HS384", "HS512":
				return []byte(jwtConf.Key), nil
			case "RS256", "RS384", "RS512":
				return jwt.ParseRSAPublicKeyFromPEM([]byte(jwtConf.Key))
			case "ES256", "ES384", "ES512":
				return jwt.ParseECPublicKeyFromPEM([]byte(jwtConf.Key))
			case "EdDSA":
				return jwt.ParseEdPublicKeyFromPEM([]byte(jwtConf.Key))
			default:
				return nil, errors.New("unsupported signature method")
			}
		}, nil
	}
}

func (i *IngestorWebServerImplemenation) apiAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// find user scopes (currently disabled, could be a nice fallback solution later)
	bearer := input.RequestValidationInput.Request.Header.Get("Authorization")
	if bearer == "" {
		return errors.New("user is not logged-in")
	}
	splitToken := strings.Split(bearer, "Bearer ")
	if len(splitToken) != 2 {
		return errors.New("invalid bearer token")
	}

	jwtToken := splitToken[1]

	var claims keycloakClaims
	_, err := jwt.ParseWithClaims(jwtToken, &claims, i.jwtKeyfunc, jwt.WithValidMethods(i.jwtSignMethods))
	if err != nil {
		return err // token is not valid (expired), likely
	}

	//kcClaims, ok := token.Claims.(keycloakClaims)
	//if !ok {
	//	return errors.New("claim extraction failed")
	//}
	fmt.Printf("here are the realm roles: \"%v\"", claims.GetRealmRoles())

	return nil // for now we accept anything that has a valid JWT token
}

func findMissingScopes(desiredScopes []string, actualScopes []string) []string {
	missingScopes := []string{}
	scopeMap := make(map[string]bool)

	for _, scope := range actualScopes {
		scopeMap[scope] = false
	}

	for _, scope := range desiredScopes {
		if _, ok := scopeMap[scope]; !ok {
			missingScopes = append(missingScopes, scope)
		}
	}

	return missingScopes
}

func missingScopesCheck(missingScopes []string, methodName string) error {
	if len(missingScopes) == 0 {
		return nil
	}
	return fmt.Errorf("missing scopes for \"%s\": %v", methodName, missingScopes)
}
