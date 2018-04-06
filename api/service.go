package api

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// rateLimit is the amount of time a requestor must wait before
// making another request
const rateLimit = time.Second * 1 // 1RPS

// ErrRateLimitExceeded is thrown when an IP exceeds the specified rate limit
var ErrRateLimitExceeded = echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded (1RPS)")

// Service contains all dependencies needed for the Tinystat service
type Service struct {
	logger  *logrus.Entry
	rateMap *rateMap
	maxApps int
	db      *gorm.DB
	cache   *cache.Cache
}

// rateMap is a a wrapper struct for performing rate-limiting
type rateMap struct {
	sync.Mutex
	ipMap map[string]time.Time // ip_action_vars... -> time
}

// NewService generates a new Service reference and return it
func NewService(logger *logrus.Logger, mysqlURL string, maxApps int, cacheExp time.Duration) (*Service, error) {
	l := logger.WithField("module", "new_service")

	// Create the MySQL Client and AutoMigrate tables
	l.Debug("Creating new MySQL Client")
	db, err := gorm.Open("mysql", mysqlURL)
	if err != nil {
		l.WithError(err).Error("Failed to connect to MySQL")
		return nil, err
	}
	db.Set("gorm:table_options", "CHARSET=utf8").AutoMigrate(&Action{})
	db.Set("gorm:table_options", "CHARSET=utf8").AutoMigrate(&App{})

	// Return the new Service
	l.Debug("Returning new service")
	return &Service{
		logger:  logger.WithField("service", "tinystat"),
		rateMap: &rateMap{ipMap: make(map[string]time.Time)},
		maxApps: maxApps,
		db:      db,
		cache:   cache.New(cacheExp, cacheExp),
	}, nil
}

// Close closes the db connection
func (s *Service) Close() error { return s.db.Close() }

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

// rateLimit returns true if the ip passed has performed too
// many requests lately
// vars should include the IP and any other variables to make
// rate limiting unique to a path
func (s *Service) rateLimit(vars ...string) bool {
	key := strings.Join(vars, "_")
	s.rateMap.Lock()
	defer s.rateMap.Unlock()

	// If this IP is in the map and it's last request
	// was within the specified ratelimit timeframe
	if last, ok := s.rateMap.ipMap[key]; ok &&
		last.After(time.Now().Add(-1*rateLimit)) {
		return true
	}

	// Set a new last request time and allow the request
	s.rateMap.ipMap[key] = time.Now()
	return false
}
