package main

import (
	"log"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := MsSQLDSN()
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	DB = db
	// AutoMigrate removed — using manual migrations in DB.
}
