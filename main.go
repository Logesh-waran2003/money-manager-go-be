package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"money-manager-go-be/database"
	"money-manager-go-be/handlers"
)

func main() {
	// Connect to Database
	database.ConnectDB()

	// Initialize Fiber app
	app := fiber.New()

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Allow all for now as requested
		AllowHeaders: "Origin, Content-Type, Accept, X-User-ID",
	}))

	// Routes
	api := app.Group("/api/v1")

	// Health Check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Sync Endpoint
	api.Post("/sync", handlers.BatchSync)

	// Manual Mapping Endpoint
	api.Post("/transactions/map", handlers.RemapTransaction)

	// AI Analysis Endpoint
	api.Get("/analyze", handlers.AnalyzeUncategorized)

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(app.Listen(":" + port))
}
