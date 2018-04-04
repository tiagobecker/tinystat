package tinystat

import (
	"time"

	"github.com/labstack/echo"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// Service contains all dependencies needed for the Tinystat service
type Service struct {
	logger  *logrus.Entry
	maxApps int
	db      *gorm.DB
	cache   *cache.Cache
}

// NewService generates a new Service reference and return it
func NewService(logger *logrus.Logger, mysqlURL string, maxApps int, cacheExp time.Duration) (*Service, error) {
	l := logger.WithField("module", "new_service")

	// Create the MySQL Client and AutoMigrate tables
	l.Debug("Creating new MySQL Client")
	db, err := gorm.Open("mysql", mysqlURL)
	if err != nil {
		return nil, err
	}
	db.Set("gorm:table_options", "CHARSET=utf8").AutoMigrate(&Action{})
	db.Set("gorm:table_options", "CHARSET=utf8").AutoMigrate(&App{})

	// Return the new Service
	l.Debug("Returning new service")
	return &Service{
		logger:  logger.WithField("service", "tinystat"),
		maxApps: maxApps,
		db:      db,
		cache:   cache.New(cacheExp, cacheExp),
	}, nil
}

// validateToken validates that the token matches the appID
// If the strictAuth value is set to true, a token MUST be valid
// If the strictAuth value is set to false, we'll refer to the users secure flag
func (s *Service) validateToken(appID string, strictAuth bool, c echo.Context) bool {
	l := s.logger.WithField("method", "validate_token")

	// Pull the token from the request
	var token string
	token = c.QueryParam("token")
	if token == "" {
		token = c.Request().Header.Get("TOKEN")
	}

	// Check the cache for a stored app/token and validate
	l.Debug("Checking cache for App")
	var app App
	if appIface, ok := s.cache.Get(appID); ok {
		app = *appIface.(*App)
	} else {
		// Attempt to retrieve the app from the DB if it couldn't be found in cache
		if err := s.db.Where(&App{ID: appID}).Find(&app).Error; err != nil {
			l.WithError(err).Error("Failed to retrieve App from DB")
			return false
		}
		// Cache the App for future
		l.Debug("Storing App in Cache")
		s.cache.SetDefault(appID, &app)
	}

	// If strict or a secure app verify token
	if strictAuth || app.StrictAuth {
		return app.Token == token
	}
	// Otherwise fuck it
	return true
}
