package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/chukiagosoftware/alpaca/vertex"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func timeoutResponse(c *gin.Context) {
	c.String(http.StatusRequestTimeout, "timeout")
}

func timeoutMiddleware() gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(60*time.Second),
		timeout.WithResponse(timeoutResponse),
	)
}

func main() {
	config, err := vertex.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Loaded config: ProjectID=%s\n, Location=%s\n, DatasetID=%s\n, IndexID=%s\n, EndpointID=%s\n, Domain=%s\n",
		config.ProjectID,
		config.Location,
		config.DatasetID,
		config.IndexID,
		config.EndpointID,
		config.EndpointPublicDomainName)

	vertex.InitTracer("alpaca-vertex-search")

	ctx := context.Background()
	// defer cancel()

	vsSvc, err := vertex.NewVertexSearchService(ctx, config)
	if err != nil {
		log.Fatal("Failed to create Vertex service:", err)
	}
	defer vsSvc.Close()

	// Setup our http server with OpenTelemetry spans
	r := gin.Default()
	r.Use(otelgin.Middleware("vertex-search"))
	r.StaticFile("/", "vertex/search/index.html")
	r.Static("/static", "vertex/search/static")

	r.Use(timeoutMiddleware())

	r.POST("/search", func(c *gin.Context) {
		SearchHandler(c, config, vsSvc)
	})

	r.GET("/ping", func(c *gin.Context) {
		// Return JSON response
		Pong(c)
	})

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
