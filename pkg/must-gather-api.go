package main

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/konveyor/forklift-must-gather-api/pkg/backend"
	"gorm.io/gorm"
)

var r *gin.Engine
var db *gorm.DB
var corsAllowedOrigins []*regexp.Regexp

func main() {
	db = backend.ConnectDB(backend.ConfigEnvOrDefault("DB_PATH", "gatherings.db"))

	// Periodical deletion of old records&archives on background
	go backend.PeriodicalCleanup(backend.ConfigEnvOrDefault("CLEANUP_MAX_AGE", "-1"), db, false)

	// Prepare gin-gonic webserver with routes and middleware
	r := setupRouter()

	// Start HTTP/HTTPS service
	if backend.ConfigEnvOrDefault("API_TLS_ENABLED", "false") == "true" {
		r.RunTLS(backend.ConfigEnvOrDefault("PORT", "8443"), backend.ConfigEnvOrDefault("API_TLS_CERTIFICATE", ""), backend.ConfigEnvOrDefault("API_TLS_KEY", ""))
	} else {
		r.Run() // PORT from ENV variable is handled inside Gin-gonic and defaults to 8080
	}
}

func setupRouter() *gin.Engine {
	// Set gin release mode if runs in the k8s cluster
	if len(backend.ConfigEnvOrDefault("KUBERNETES_SERVICE_HOST", "")) > 0 {
		gin.SetMode(gin.ReleaseMode)
	}

	// Gin setup
	r = gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(200, "must-gather-rest-wrapper - for API see https://github.com/konveyor/forklift-must-gather-api#README")
	})

	// Setup middleware to validate bearer auth tokens against the cluster
	r.Use(backend.DefaultAuth.Permit)

	// Prepare CORS
	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{"Authorization", "Origin"},
		AllowOriginFunc:  corsAllow,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup routes for must-gather functions
	r.POST("/must-gather", triggerGathering)
	r.GET("/must-gather", listGatherings) // good at least during development and testing, real user should know gathering ID
	r.GET("/must-gather/:id", getGathering)
	r.GET("/must-gather/:id/data", getGatheringArchive)

	return r
}

func triggerGathering(c *gin.Context) {
	var gathering backend.Gathering
	if err := c.Bind(&gathering); err == nil {
		gathering.Status = "new"
		if gathering.Image == "" {
			gathering.Image = backend.ConfigEnvOrDefault("DEFAULT_IMAGE", "quay.io/konveyor/forklift-must-gather") // default image configurable via OS ENV vars
		}
		if gathering.Timeout == "" {
			gathering.Timeout = backend.ConfigEnvOrDefault("TIMEOUT", "20m") // default timeout for must-gather execution
		}
		gathering.AuthToken = strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer ")
		// TODO: Check the token or just pass to the commandline? but always satinize to not explode token to multiple commands (steal previous executions data or tokens)
		db.Create(&gathering)
		c.JSON(201, gathering)
		go backend.MustGatherExec(&gathering, db, backend.ConfigEnvOrDefault("ARCHIVE_FILENAME", "must-gather.tar.gz"))
	} else {
		log.Printf("Error creating gathering: %v", err)
		c.JSON(400, "create gathering error")
	}
}

// TODO filter results by authtoken
func listGatherings(c *gin.Context) {
	var Gatherings []backend.Gathering
	db.Find(&Gatherings)
	c.JSON(200, Gatherings)
}

func getGathering(c *gin.Context) {
	var gathering backend.Gathering
	db.First(&gathering, "id = ?", c.Param("id")) // ID (uint) lookup - safe way to handle possible string input to not interpret it as a query
	if gathering.ID != 0 {
		c.JSON(200, gathering)
	} else {
		db.Last(&gathering, "custom_name = ?", c.Param("id")) // Fallback to CustomName (string) lookup - returned the newest/last matching record
		if gathering.ID != 0 {
			c.JSON(200, gathering)
		} else {
			// Return empty gathering with 404 code if not found
			c.JSON(404, gathering)
		}
	}
}

func getGatheringArchive(c *gin.Context) {
	var gathering backend.Gathering
	db.First(&gathering, "id = ?", c.Param("id"))
	if gathering.ID != 0 && gathering.Status == "completed" {
		c.FileAttachment(gathering.ArchivePath, gathering.ArchiveName)
	} else {
		c.String(404, "")
	}
}

// CORS functions
func corsAllow(origin string) bool {
	for _, expr := range corsAllowedOrigins {
		if expr.MatchString(origin) {
			return true
		}
	}

	return false
}

//func corsBuildOrigins() {
//	corsAllowedOrigins = []*regexp.Regexp{}
//	for _, r := range ConfigEnvOrDefault("CORS_ALLOWED_ORIGINS", "") {
//		expr, err := regexp.Compile(r)
//		if err != nil {
//			continue
//		}
//		corsAllowedOrigins = append(corsAllowedOrigins, expr)
//	}
//}
