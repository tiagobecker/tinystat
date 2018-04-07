package api

import (
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// Service contains all dependencies needed for the Tinystat service
type Service struct {
	logger  *logrus.Entry
	appID   string // tinystatAppID
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
func NewService(logger *logrus.Logger, tinystatAppID string, mysqlURL string, maxApps int, cacheExp time.Duration) (*Service, error) {
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
		appID:   tinystatAppID,
		rateMap: &rateMap{ipMap: make(map[string]time.Time)},
		maxApps: maxApps,
		db:      db,
		cache:   cache.New(cacheExp, cacheExp),
	}, nil
}

// Close closes the db connection
func (s *Service) Close() error { return s.db.Close() }
