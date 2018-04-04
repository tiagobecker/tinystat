package tinystat

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
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
	ID        string    `json:"id,omitempty" gorm:"type:varchar(10);primary_key;unique_index"`
	Name      string    `json:"name,omitempty" gorm:"type:varchar(100);not null"`
	Token     string    `json:"token,omitempty" gorm:"type:varchar(32);not null"`
	Secure    bool      `json:"secure,omitempty" gorm:"type:bool;not null"`
	IP        string    `json:"ip,omitempty" gorm:"type:varchar(40);index;not null"`
	CreatedAt time.Time `json:"createdAt,omitempty" sql:"index"`
}

// CreateApp creates a new application and stores it in the database
// Endpoint: /app/create/:name
func (s *Service) CreateApp(c echo.Context) error {
	l := s.logger.WithField("method", "create_app")
	l.Debug("Received new CreateApp request")

	// Check if maximum apps has been exceeded
	l.Debug("Verifying the IP hasn't exceeded max Apps")
	ip := c.RealIP()
	apps, err := s.currentApps(ip)
	if err != nil {
		return ErrAppCountRetrievalFailure
	}
	if apps >= s.maxApps {
		return ErrMaxAppsExceeded
	}

	// Generates an AppID UUID and a Token UUID
	l.Debug("Generating new App UUIDs")
	name := c.Param("name")
	secure, _ := strconv.ParseBool(c.QueryParam("secure"))
	appID := newAppID()
	token := newUUID()
	l = l.WithFields(map[string]interface{}{
		"name": name, "app_id": appID, "token": token})

	// Create a new App from the generated UUIDs
	l.Debug("Generating new App")
	newApp := &App{
		ID:        appID,
		Name:      name,
		Token:     token,
		IP:        ip,
		Secure:    secure,
		CreatedAt: time.Now(), // Use the servers current time
	}

	// Insert the new App in the DB
	l.Debug("Storing new App in DB")
	if err := s.db.Create(newApp).Error; err != nil {
		l.WithError(err).Error("Failed to create new App in DB")
		return ErrAppStoreFailure
	}

	// Cache the app for future actions
	l.Debug("Storing App in Cache")
	s.cache.SetDefault(appID, newApp.Token)

	// Clear fields we don't want to expose on the output
	newApp.IP = ""
	newApp.CreatedAt = time.Time{}

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
