package main

import (
	"math/rand"
	"strings"
	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sdwolfe32/tinystat/api"
	"github.com/sdwolfe32/tinystat/config"
	"github.com/sirupsen/logrus"
)

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate the logger and set the formatter
	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	// Configure the logger based on the environment
	if strings.Contains(config.Env, "prod") {
		logger.Formatter = new(logrus.JSONFormatter)
		logger.Level = logrus.InfoLevel
	}
	l := logger.WithField("port", config.Port)

	// Create the tinystat service
	l.Info("Generating all Tinystat dependencies")
	s, err := api.NewService(logger, config.MysqlURL, config.MaxAppsPerIP, time.Hour*24)
	if err != nil {
		l.WithError(err).Fatalln("Failed to generate Tinystat service")
	}
	defer s.Close()

	// Generate the router
	l.Info("Generating router and middleware")
	e := echo.New()
	e.Use(middleware.Recover())

	// Bind all handlers to the router
	l.Info("Binding API endpoints to the router")
	e.POST("/v1/app/create/:name", s.CreateApp)
	e.POST("/v1/app/:app_id/action/:action/create/:count", s.CreateAction)
	e.GET("/v1/app/:app_id/action/:action/count", s.ActionSummary)
	e.GET("/v1/app/:app_id/action/:action/count/:duration", s.ActionCount)
	e.GET("/v1/stats", s.Stats)

	// Host static demo pages if configured to do so
	if config.ServeWeb {
		l.Info("Serving web UI on index")
		e.Static("/", "web")
		e.Static("/assets", "web/assets")
	}

	// Bind the handlers and listen for requests
	l.Info("Listening for requests")
	e.Logger.Fatal(e.Start(":" + config.Port))
}
