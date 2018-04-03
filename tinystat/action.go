package tinystat

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
)

var (
	// ErrInvalidToken is thrown when a request fails to be authenticated
	ErrInvalidToken = echo.NewHTTPError(http.StatusUnauthorized, "Failed to validate token")
	// ErrIncrementFailure is thrown when we fail to increment an action
	ErrIncrementFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to increment Action count")
	// ErrParseDurationFailure is thrown when we fail to parse a duration
	ErrParseDurationFailure = echo.NewHTTPError(http.StatusBadRequest, "Failed to parse duration")
	// ErrCountSumFailure is thrown when we fail to retrieve the Action count sum
	ErrCountSumFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve Action count")
)

// Action is any action that can be stored with a timestamp
type Action struct {
	ID        string    `gorm:"primary_key" gorm:"primary_key;unique_index"`
	AppID     string    `sql:"index" gorm:"type:varchar(10);not null"`
	Action    string    `sql:"index" gorm:"type:varchar(100);not null"`
	Count     int64     `gorm:"not null"`
	Timestamp time.Time `sql:"index"`
}

// CountAction increments the database value for the pas
// Endpoint: /action/:app_id/:metric/create?token=:token
func (s *Service) CountAction(c echo.Context) error {
	l := s.logger.WithField("method", "count_action")
	l.Debug("Received new CountAction request")

	// Decode the request variables
	appID := c.Param("app_id")
	action := c.Param("action")
	token := c.QueryParam("token")
	l = l.WithFields(map[string]interface{}{
		"app_id": appID,
		"action": action,
		"token":  token,
	})

	// Validate the token on the request
	l.Debug("Validating the passed token")
	if valid := s.validateToken(appID, token); !valid {
		l.Error("Failed to validate token")
		return ErrInvalidToken
	}

	// Get the current day and use it as a timestamp
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Store the new action in the database
	l.Debug("Incrementing Action count in DB")
	if err := s.incrementAction(appID, action, today); err != nil {
		l.WithError(err).Error("Failed to increment Action count")
		return ErrIncrementFailure
	}

	// Return an Status OK
	l.Debug("Returning successful CountAction response")
	return c.JSON(http.StatusOK, nil)
}

// GetActionCount retrieves the count of actions for an app in the
// passed duration. Duration should match the same formatting as
// https://golang.org/pkg/time/#ParseDuration
// Endpoint: /action/:app_id/:metric/count/:duration
func (s *Service) GetActionCount(c echo.Context) error {
	l := s.logger.WithField("method", "get_action_count")
	l.Debug("Received new GetActionCount request")

	// Decode the request variables
	appID := c.Param("app_id")
	action := c.Param("action")
	token := c.QueryParam("token")
	duration := c.Param("duration")
	l = l.WithFields(map[string]interface{}{
		"app_id":   appID,
		"action":   action,
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
	count, err := s.actionSum(appID, action, startTime)
	if err != nil {
		l.WithError(err).Error("Failed to retrieve Action sum")
		return ErrCountSumFailure
	}

	// Return an Status OK
	l.Debug("Returning successful GetActionCount response")
	return c.JSON(http.StatusOK, count)
}

// incrementAction will attempt to increment the count value
// for an existing Action record for the day. If one doesn't exist
// a new one will be created with with a count of 1
func (s *Service) incrementAction(appID, action string, timestamp time.Time) error {
	key := generateKey(appID, action, timestamp)
	return s.db.Exec(`INSERT INTO actions(id, app_id, action, count, timestamp) VALUES(?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE count = count + 1`,
		key, appID, action, 1, timestamp).Error
}

// generateKey generates and returns a unique, deterministic key
// for an action
func generateKey(appID, action string, timestamp time.Time) string {
	keySlice := []string{appID, action, timestamp.String()}
	keyStr := strings.Join(keySlice, "_")
	return fmt.Sprintf("%x", md5.Sum([]byte(keyStr)))
}

// SumResult represents a sum query result
type SumResult struct{ Total int64 }

// actionSum will attempt to retrieve all daily actions and SUM
// them all to retrieve the total number of actions
func (s *Service) actionSum(appID, action string, startTime time.Time) (int64, error) {
	var res SumResult
	return res.Total, s.db.
		Table("actions").
		Select("sum(count) as total").
		Where("app_id = ?", appID).
		Where("action = ?", action).
		Where("timestamp > ?", startTime).
		Scan(&res).Error
}
