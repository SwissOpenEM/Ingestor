package webserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
)

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
	// if auth is disabled, return immediately
	if i.taskQueue.Config.Auth.Disable {
		return nil
	}

	// jwt authentication
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

	// RBAC
	foundRoles := claims.GetResourceRolesByKey(i.taskQueue.Config.Auth.JWTConf.ClientID)

	// if admin, accept
	if slices.Contains(foundRoles, i.scopeToRoleMap["admin"]) {
		return nil
	}

	// check for missing roles
	requiredRoles := i.mapScopesToRoles(input.Scopes)
	missingRoles := findMissingRoles(requiredRoles, foundRoles)
	return missingRolesCheck(missingRoles, input.RequestValidationInput.Request.Method+" "+input.RequestValidationInput.Request.RequestURI)
}

func (i *IngestorWebServerImplemenation) mapScopesToRoles(scopes []string) []string {
	var roles []string
	for _, role := range scopes {
		if val, ok := i.scopeToRoleMap[role]; ok {
			roles = append(roles, val)
		}
	}
	return roles
}

func createScopeToRoleMap(conf core.RBACConf) (map[string]string, error) {
	scopeMap := make(map[string]string)

	// check config
	if conf.AdminRole == "" {
		return nil, errors.New("AdminRole is not set in config")
	}
	if conf.CreateModifyTasksRole == "" {
		return nil, errors.New("CreateModifyTasksRole is not set in config")
	}
	if conf.ViewTasksRole == "" {
		return nil, errors.New("ViewTasksRole is not set in config")
	}

	// map the roles to scopes
	scopeMap["admin"] = conf.AdminRole
	scopeMap["ingestor_write"] = conf.CreateModifyTasksRole
	scopeMap["ingestor_read"] = conf.ViewTasksRole
	return scopeMap, nil
}

func findMissingRoles(requiredRoles []string, foundRoles []string) []string {
	missingRoles := []string{}
	roleMap := make(map[string]bool)

	for _, role := range foundRoles {
		roleMap[role] = false
	}

	for _, role := range requiredRoles {
		if _, ok := roleMap[role]; !ok {
			missingRoles = append(missingRoles, role)
		}
	}

	return missingRoles
}

func missingRolesCheck(missingRoles []string, methodName string) error {
	if len(missingRoles) == 0 {
		return nil
	}
	return fmt.Errorf("missing roles for \"%s\": %v", methodName, missingRoles)
}
