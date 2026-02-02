package main

import (
	"log"

	"github.com/searchktools/fast-server/app"
	"github.com/searchktools/fast-server/config"
	"github.com/searchktools/fast-server/core/http"
)

func main() {
	// Create configuration
	cfg := config.New()

	// Create application
	application := app.New(cfg)

	// Get engine for route registration
	engine := application.Engine()

	// Simple text response
	engine.GET("/", func(ctx http.Context) {
		ctx.String(200, "Welcome to Fast Server!")
	})

	// JSON response
	engine.GET("/api/status", func(ctx http.Context) {
		ctx.JSON(200, map[string]interface{}{
			"status":  "ok",
			"version": "1.0.0",
			"server":  "fast-server",
		})
	})

	// Path parameters
	engine.GET("/api/users/:id", func(ctx http.Context) {
		id := ctx.Param("id")
		ctx.JSON(200, map[string]string{
			"user_id": id,
			"name":    "John Doe",
		})
	})

	// Query parameters
	engine.GET("/api/search", func(ctx http.Context) {
		query := ctx.Query("q")
		page := ctx.Query("page")
		
		ctx.JSON(200, map[string]string{
			"query": query,
			"page":  page,
		})
	})

	// POST with body
	engine.POST("/api/users", func(ctx http.Context) {
		// In a real app, you would parse the body here
		ctx.JSON(201, map[string]string{
			"message": "User created",
		})
	})

	// Start the server
	log.Printf("Starting Fast Server...")
	application.Run()
}
