package webserver

import (
	"fmt"
	"net"
	"net/http"
	"os"

	docs "github.com/SwissOpenEM/Ingestor/docs"
	cors "github.com/gin-contrib/cors"
	gin "github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

	// The swagger docs have to come before the default handlers
	docs.SwaggerInfo.BasePath = r.BasePath()
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	r.Use(middleware.OapiRequestValidator(swagger))

	RegisterHandlers(r, ingestor)

	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", fmt.Sprint(port)),
	}
	return s
}
