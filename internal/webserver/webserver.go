package webserver

import (
	"embed"
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/SwissOpenEM/Ingestor/internal/webserver/randomfuncs"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/getkin/kin-openapi/openapi3filter"
	cors "github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	gin "github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Copy the openapi specs to local folder so it can be embedded in order to statically serve it
//go:generate cp ../../api/openapi.yaml ./openapi.yaml

//go:embed openapi.yaml
var swaggerYAML embed.FS

func NewIngesterServer(ingestor *IngestorWebServerImplemenation, port int) *http.Server {
	swagger, err := GetSwagger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	// This is how you set up a basic gin router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Accept"},
	}))

	// The route here needs to match the embedded file.
	// The openapi specs are serve statically so that the swagger ui can refer to it.
	r.GET("/openapi.yaml", func(c *gin.Context) {
		http.FileServer(http.FS(swaggerYAML)).ServeHTTP(c.Writer, c.Request)
	})

	// The swagger docs have to come before the default handlers
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler, ginSwagger.URL("/openapi.yaml")))

	// setup auth session store
	authKey, err := randomfuncs.GenerateRandomByteSlice(64) // authentication key
	if err != nil {
		panic(err)
	}
	encKey, err := randomfuncs.GenerateRandomByteSlice(32) // encryption key
	if err != nil {
		panic(err)
	}
	store := cookie.NewStore(authKey, encKey)
	store.Options(sessions.Options{
		HttpOnly: true,
	})

	// register types to be stored in cookies
	gob.Register(oidc.UserInfo{})

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	r.Use(
		sessions.SessionsMany([]string{"auth", "user"}, store),
		middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: ingestor.apiAuthFunc,
			},
		}),
	)
	RegisterHandlers(r, NewStrictHandler(ingestor, []StrictMiddlewareFunc{}))

	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", fmt.Sprint(port)),
	}
	return s
}
