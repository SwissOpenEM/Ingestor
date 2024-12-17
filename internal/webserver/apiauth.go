package webserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

func initKeyfunc(jwtConf wsconfig.JWTConf) (jwt.Keyfunc, error) {
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
	if i.taskQueue.Config.WebServer.AuthConf.Disable {
		return nil
	}

	// get session
	ginCtx, ok := ctx.Value(ginmiddleware.GinContextKey).(*gin.Context)
	if !ok {
		return errors.New("can't get gin context")
	}
	userSession := sessions.DefaultMany(ginCtx, "user")

	// check expiry
	expiresAtString, ok := userSession.Get("expires_at").(string)
	if !ok {
		return errors.New("login session has expired")
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtString)
	if err != nil {
		return fmt.Errorf("can't parse \"expire by\" string: %s", err.Error())
	}
	if expiresAt.Before(time.Now()) {
		userSession.Options(sessions.Options{
			MaxAge: -1,
		})
		_ = userSession.Save() // ignore error
		return errors.New("login session has expired")
	}

	// RBAC
	foundRoles, ok := userSession.Get("roles").([]string)
	if !ok {
		return errors.New("can't extract roles")
	}
	if slices.Contains(foundRoles, i.scopeToRoleMap["admin"]) {
		return nil // if admin, accept by default
	}
	requiredRoles := i.mapScopesToRoles(input.Scopes)
	missingRoles := findMissingRoles(requiredRoles, foundRoles)
	return missingRolesCheck(missingRoles, input.RequestValidationInput.Request.Method+" "+input.RequestValidationInput.Request.RequestURI)
}

func parseKeycloakJWTToken(token string, keyfunc jwt.Keyfunc, signMethods []string) (keycloakClaims, error) {
	var claims keycloakClaims
	_, err := jwt.ParseWithClaims(token, &claims, keyfunc, jwt.WithValidMethods(signMethods))
	if err != nil {
		return keycloakClaims{}, err // token is not valid (expired), likely
	}
	return claims, nil
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

func createScopeToRoleMap(conf wsconfig.RBACConf) (map[string]string, error) {
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
