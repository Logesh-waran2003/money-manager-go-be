package database

import (
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"money-manager-go-be/models"
)

// DB instance
var DB *gorm.DB

// ConnectDB connects to the database
func ConnectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}
	if !strings.Contains(dsn, "sslmode") {
		dsn += "?sslmode=require" // Fixes Supabase connection refusal
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	log.Println("Connected to database successfully")

	// Auto Migrate
	log.Println("Running migrations...")
	err = DB.AutoMigrate(&models.User{}, &models.Transaction{}, &models.CategoryRule{})
	if err != nil {
		log.Fatal("Failed to migrate database. \n", err)
	}
	log.Println("Database migrated successfully")
}
