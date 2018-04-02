package config

import (
	"os"
	"strings"
)

var (
	// Port is the port the server will run on
	Port = getEnv("PORT", "8080")
	// Env is the environment (development, production)
	Env = strings.ToLower(getEnv("ENVIRONMENT", "development"))
	// MysqlURL is the URL used to access the MySQL database
	MysqlURL = strings.ToLower(getEnv("MYSQL_URL", ""))
)

// getEnv retrieves variables from the environment and falls back
// to a passed fallback variable if it isn't already set
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
