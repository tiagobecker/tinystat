package tinystat

import (
	"net/http"
	"time"

	"github.com/labstack/echo"
)

var (
	// ErrInvalidToken is thrown when a request fails to be authenticated
	ErrInvalidToken = echo.NewHTTPError(http.StatusUnauthorized, "Failed to validate token")
	// ErrCreationFailure is thrown when we fail to store a new Action
	ErrCreationFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to create new Action")
	// ErrParseDurationFailure is thrown when we fail to parse a duration
	ErrParseDurationFailure = echo.NewHTTPError(http.StatusBadRequest, "Failed to parse duration")
)

// Action is any action that can be stored with a timestamp
type Action struct {
	ID        uint      `gorm:"primary_key"`
	AppID     string    `sql:"index"`
	CreatedAt time.Time `sql:"index"`
}

// CreateAction creates a new Action and stores it in the database
// Endpoint: /action/:app_id/create?token=:token
func (s *Service) CreateAction(c echo.Context) error {
	l := s.logger.WithField("method", "action")
	l.Debug("Received new Action request")

	// Decode the request variables
	appID := c.Param("app_id")
	token := c.QueryParam("token")
	l = l.WithFields(map[string]interface{}{
		"app_id": appID,
		"token":  token,
	})

	// Validate the token on the request
	l.Debug("Validating the passed token")
	if valid := s.validateToken(appID, token); !valid {
		l.Error("Failed to validate token")
		return ErrInvalidToken
	}

	// Create a new Action
	l.Debug("Generating new Action")
	newAction := &Action{AppID: appID}

	// Store the new action in the database
	if err := s.db.Create(newAction).Error; err != nil {
		l.WithError(err).Error("Failed to create Action in DB")
		return ErrCreationFailure
	}
	return c.JSON(http.StatusOK, nil)
}

// GetActionCount retrieves the count of actions for an app in the
// passed duration. Duration should match the same formatting as
// https://golang.org/pkg/time/#ParseDuration
// Endpoint: /action/:app_id/count/:duration
func (s *Service) GetActionCount(c echo.Context) error {
	l := s.logger.WithField("method", "count")
	l.Debug("Received new Count request")

	// Decode the request variables
	appID := c.Param("app_id")
	token := c.QueryParam("token")
	duration := c.Param("duration")
	l = l.WithFields(map[string]interface{}{
		"app_id":   appID,
		"token":    token,
		"duration": duration,
	})

	// Validate the token on the request
	l.Debug("Validating the passed token")
	if valid := s.validateToken(appID, token); !valid {
		l.Error("Failed to validate token")
		return ErrInvalidToken
	}

	// Parse the duration passed
	l.Debug("Parsing the requested duration")
	dur, err := time.ParseDuration(duration)
	if err != nil {
		l.WithError(err).Error("Failed to parse duration")
		return ErrParseDurationFailure
	}

	// Calculate the starting bound
	startTime := time.Now().Add(-1 * dur)

	// Retrieve the action count from the DB and return
	l.Debug("Retrieve the count of Actions from the DB")
	var count int64
	if err := s.db.Model(&Action{}).
		Where("app_id = ?", appID).
		Where("created_at > ?", startTime).
		Count(&count).Error; err != nil {
		l.WithError(err).Error("Failed to retrieve Action count from DB")
		return err
	}
	return c.JSON(http.StatusOK, count)
}
