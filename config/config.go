package config

import (
	"os"
	"strconv"
	"strings"
)

var (
	// Port is the port the server will run on
	Port = getEnv("PORT", "8080")
	// Env is the environment (development, production)
	Env = strings.ToLower(getEnv("ENVIRONMENT", "development"))
	// MysqlURL is the URL used to access the MySQL database
	MysqlURL = getEnv("MYSQL_URL", "")
	// ServeWeb defines if the web static site should be served
	ServeWeb, _ = strconv.ParseBool(getEnv("SERVE_WEB", "false"))
	// MaxAppsPerIP is the number of Apps each IP is allowed to have
	MaxAppsPerIP, _ = strconv.Atoi(getEnv("MAX_APPS_PER_IP", "5"))
	// TinystatAppID is the App ID used with Tinystat
	TinystatAppID = getEnv("TINYSTAT_APP_ID", "")
	// TinystatToken is the Token used to authenticate Tinystat requests
	TinystatToken = getEnv("TINYSTAT_TOKEN", "")
)

// getEnv retrieves variables from the environment and falls back
// to a passed fallback variable if it isn't already set
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
