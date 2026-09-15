package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"foodrescue-api/internal/config"
)

var DB *sql.DB

func Connect() {
	var dsn string

	if config.AppConfig.DBDSN != "" {
		dsn = config.AppConfig.DBDSN
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.AppConfig.DBUser,
			config.AppConfig.DBPassword,
			config.AppConfig.DBHost,
			config.AppConfig.DBPort,
			config.AppConfig.DBName,
		)
	}

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connected successfully")
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}