package database

import (
	"context"
	"fmt"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
)

// SchemaConnections holds database connections for different schemas
var SchemaConnections = make(map[string]*gorm.DB)

type SchemaConfig struct {
	SchemaName string
}

func ConnectWithSchema(schema string, lc fx.Lifecycle, models ...interface{}) (*gorm.DB, error) {
	if conn, exists := SchemaConnections[schema]; exists {
		return conn, nil
	}

	baseURL := getBaseURL()
	db, err := gorm.Open(postgres.Open(baseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	if err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)).Error; err != nil {
		return nil, fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	// Migrate
	if err := db.Exec(fmt.Sprintf("SET search_path TO %s", schema)).Error; err != nil {

		return nil, fmt.Errorf("schema %s does not exist: %w", schema, err)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			return nil, fmt.Errorf("failed to migrate schema %s: %w", schema, err)
		}
	}

	// Register DB close hook
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			log.Println("Closing DB connection...")
			return sqlDB.Close()
		},
	})

	SchemaConnections[schema] = db
	log.Printf("Connected to schema: %s", schema)
	return db, nil
}

func getBaseURL() string {
	if os.Getenv("ENV") == "staging" {
		return os.Getenv("RDS_CONNECTION")
	}
	return os.Getenv("DATABASE_URL")
}

// Paginate remains unchanged
func Paginate(page int, pageSize int) func(db *gorm.DB) *gorm.DB {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset((page - 1) * pageSize).Limit(pageSize)
	}
}
