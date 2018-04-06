package tinystat

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
	"github.com/sdwolfe32/tinystat/client"
)

var (
	// ErrAppCountRetrievalFailure is thrown when we fail to retrieve the count of Apps for an IP
	ErrAppCountRetrievalFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve IPs current App count")
	// ErrMaxAppsExceeded is thrown when the requester has exceeded the maximum number of Apps allowed
	ErrMaxAppsExceeded = echo.NewHTTPError(http.StatusForbidden, "The maximum Apps for this IP has been exceeded")
	// ErrAppStoreFailure is thrown when there is an error storing a new App in the DB
	ErrAppStoreFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to store new App in DB")
)

// App is an application that we will count actions for
type App struct {
	ID         string    `json:"id" gorm:"type:varchar(10);primary_key;unique_index"`
	Name       string    `json:"name" gorm:"type:varchar(100);not null"`
	Token      string    `json:"token" gorm:"type:varchar(32);not null"`
	StrictAuth bool      `json:"strictAuth" gorm:"type:bool;not null"`
	IP         string    `json:"ip" gorm:"type:varchar(40);index;not null"`
	CreatedAt  time.Time `json:"createdAt" sql:"index"`
}

// CreateApp creates a new application and stores it in the database
// Endpoint: /app/create/:name
func (s *Service) CreateApp(c echo.Context) error {
	l := s.logger.WithField("method", "create_app")
	l.Debug("Received new CreateApp request")

	// Decode the request variables
	ip := c.RealIP()
	name := c.Param("name")
	strictAuth, _ := strconv.ParseBool(c.QueryParam("strict_auth"))
	l = l.WithFields(map[string]interface{}{
		"name": name, "strict_auth": strictAuth})

	// Check rate limit
	l.Debug("Checking rate limit")
	if s.rateLimit(c.RealIP()) {
		l.Error("Rate limit exceeded")
		return ErrRateLimitExceeded
	}

	// Generates an AppID UUID and a Token UUID
	l.Debug("Generating new App UUIDs")
	appID := newAppID()
	token := newUUID()
	l = l.WithFields(map[string]interface{}{
		"app_id": appID, "token": token})

	// Check if maximum apps has been exceeded
	l.Debug("Verifying the IP hasn't exceeded max Apps")
	apps, err := s.currentApps(ip)
	if err != nil {
		l.WithError(err).Error("Failed to get current apps for IP")
		return ErrAppCountRetrievalFailure
	}
	if apps >= s.maxApps {
		return ErrMaxAppsExceeded
	}

	// Create a new App from the generated UUIDs
	l.Debug("Generating new App")
	newApp := &App{
		ID:         appID,
		Name:       name,
		Token:      token,
		IP:         ip,
		StrictAuth: strictAuth,
		CreatedAt:  time.Now(), // Use the servers current time
	}

	// Insert the new App in the DB
	l.Debug("Storing new App in DB")
	if err := s.db.Create(newApp).Error; err != nil {
		l.WithError(err).Error("Failed to create new App in DB")
		return ErrAppStoreFailure
	}

	// Cache the app for future actions
	l.Debug("Storing App in Cache")
	s.cache.SetDefault(appID, newApp)

	// Report the successful create-app to ourselves
	client.CreateAction("create-app")

	// Return the newly generated App
	l.Debug("Returning newly generated/stored App")
	return c.JSON(http.StatusOK, newApp)
}

// currentApps returns the number of apps an IP has created
func (s *Service) currentApps(ip string) (int, error) {
	var count int
	return count, s.db.Model(&App{}).
		Where(&App{IP: ip}).Count(&count).Error
}

// newAppID generates the first part of a new V4 UUID
func newAppID() string {
	return newUUID()[0:10]
}

// newUUID generates a new randomly generated V4 UUID
func newUUID() string {
	u, _ := uuid.NewV4()
	return strings.Replace(u.String(), "-", "", -1)
}
