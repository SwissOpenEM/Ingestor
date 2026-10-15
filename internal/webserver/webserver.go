package webserver

import (
	"embed"
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
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

	"github.com/gin-contrib/slog"
)

//go:embed openapi.yaml
var swaggerYAML embed.FS

// Add the X-Forwarded-Prefix header to the redirect URL if it exists. This is used when the ingestor is behind a reverse proxy that strips a prefix from the URL path.
func addForwardedPrefix(c *gin.Context, target string) string{
	prefix := c.GetHeader("X-Forwarded-Prefix")
	// Use 'Join' to collapse `//`, ensuring we don't redirect to an external site.
	return path.Join("/", prefix, target)
}

func NewIngestorServer(ingestor *IngestorWebServerImplemenation, port int) *http.Server {
	swagger, err := GetSwagger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil
	// This is how you set up a basic gin router
	r := gin.New()

	r.Use(
		slog.SetLogger(slog.WithSkipPath([]string{"/version", "/health"}),
			slog.WithRequestHeader(false),
		))
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{ingestor.frontend.origin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// The route here needs to match the embedded file.
	// The openapi specs are serve statically so that the swagger ui can refer to it.
	r.GET("/openapi.yaml", func(c *gin.Context) {
		http.FileServer(http.FS(swaggerYAML)).ServeHTTP(c.Writer, c.Request)
	})

	// The swagger docs have to come before the default handlers
	swaggerHandler := ginSwagger.WrapHandler(swaggerfiles.Handler, ginSwagger.URL("../openapi.yaml"))
	r.GET("/docs/*any", func(c *gin.Context) {
		if c.Param("any") == "/" {
			// redirect so it still works behind a prefix-stripping proxy
			c.Redirect(http.StatusFound, addForwardedPrefix(c, "/docs/index.html"))
			return
		}
		swaggerHandler(c)
	})

	// Redirect / to swagger
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, addForwardedPrefix(c, "/docs/index.html"))
	})

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
		SameSite: http.SameSiteNoneMode,
	})

	// register types to be stored in cookies
	gob.Register(oidc.UserInfo{})

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	sessionsToCreate := []string{"auth", "user"}
	if ingestor.taskQueue.GetTransferMethod() == transfertask.TransferGlobus {
		sessionsToCreate = append(sessionsToCreate, "globus")
	}
	r.Use(
		sessions.SessionsMany(sessionsToCreate, store),
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
