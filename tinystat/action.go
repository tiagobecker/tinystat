package tinystat

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrInvalidToken is thrown when a request fails to be authenticated
	ErrInvalidToken = echo.NewHTTPError(http.StatusUnauthorized, "Failed to validate token")
	// ErrIncrementFailure is thrown when we fail to increment an action
	ErrIncrementFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to increment Action count")
	// ErrParseCountFailure is thrown when we fail to parse the count
	ErrParseCountFailure = echo.NewHTTPError(http.StatusBadRequest, "Failed to parse count")
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

// CreateAction increments the database value for the pas
// Endpoint: /action/:app_id/:metric/create?token=:token
func (s *Service) CreateAction(c echo.Context) error {
	l := s.logger.WithField("method", "create_action")
	l.Debug("Received new CreateAction request")

	// Check rate limit
	l.Debug("Checking rate limit")
	if s.rateLimit(c.RealIP()) {
		return ErrRateLimitExceeded
	}

	// Decode the request variables
	appID := c.Param("app_id")
	action := c.Param("action")
	countQuery := c.QueryParam("count")
	count := 1
	if countQuery != "" {
		l.Debug("Decoding count query")
		c, err := strconv.Atoi(countQuery)
		if err != nil {
			return ErrParseCountFailure
		}
		count = c
	}
	l = l.WithFields(map[string]interface{}{
		"app_id": appID, "action": action, "count": count})

	// Validate the token on the request
	l.Debug("Validating the passed token")
	if valid := s.validateToken(appID, true, c); !valid {
		l.Error("Failed to validate token")
		return ErrInvalidToken
	}

	// Store the new action in the database
	l.Debug("Incrementing Action count in DB")
	if err := s.incrementAction(appID, action, count); err != nil {
		l.WithError(err).Error("Failed to increment Action count")
		return ErrIncrementFailure
	}

	// Return a Status OK
	l.Debug("Returning successful CreateAction response")
	return c.JSON(http.StatusOK, nil)
}

// GetActionCount retrieves the count of actions for an app in the
// passed duration. Duration should match the same formatting as
// https://golang.org/pkg/time/#ParseDuration
// Endpoint: /action/:app_id/action/:action/count/:duration
func (s *Service) GetActionCount(c echo.Context) error {
	l := s.logger.WithField("method", "get_action_count")
	l.Debug("Received new GetActionCount request")

	// Check rate limit
	l.Debug("Checking rate limit")
	if s.rateLimit(c.RealIP()) {
		return ErrRateLimitExceeded
	}

	// Decode the request variables
	appID := c.Param("app_id")
	action := c.Param("action")
	duration := c.Param("duration")
	l = l.WithFields(map[string]interface{}{
		"app_id": appID, "action": action, "duration": duration})

	// Validate the token on the request
	l.Debug("Validating the passed token")
	if valid := s.validateToken(appID, false, c); !valid {
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

	now := time.Now() // Get the current time for calculating in actionSum

	// Retrieve the action count from the DB and return
	l.Debug("Retrieve the count of Actions from the DB")
	var count int64
	if err := s.actionSum(appID, action, now.Add(-1*dur), &count); err != nil {
		l.WithError(err).Error("Failed to retrieve Action sum")
		return ErrCountSumFailure
	}

	// Return an Status OK
	l.Debug("Returning successful GetActionCount response")
	return c.JSON(http.StatusOK, count)
}

// ActionSummary contains a summary of actions over several passed intervals
type ActionSummary struct {
	Hour  int64 `json:"hour"`
	Day   int64 `json:"day"`
	Week  int64 `json:"week"`
	Month int64 `json:"month"`
	Year  int64 `json:"year"`
}

// GetActionSummary retrieves all most recent counts of actions for an app and
// organizes it into a summary. Duration should match the same formatting as
// https://golang.org/pkg/time/#ParseDuration
// Endpoint: /action/:app_id/action/:action/summary
func (s *Service) GetActionSummary(c echo.Context) error {
	l := s.logger.WithField("method", "get_action_summary")
	l.Debug("Received new GetActionSummary request")

	// Check rate limit
	l.Debug("Checking rate limit")
	if s.rateLimit(c.RealIP()) {
		return ErrRateLimitExceeded
	}

	// Decode the request variables
	appID := c.Param("app_id")
	action := c.Param("action")
	l = l.WithFields(map[string]interface{}{
		"app_id": appID, "action": action})

	// Validate the token on the request
	l.Debug("Validating the passed token")
	if valid := s.validateToken(appID, false, c); !valid {
		l.Error("Failed to validate token")
		return ErrInvalidToken
	}

	now := time.Now() // Get the current time for calculating in actionSum

	// Retrieve all count values and place them on the ActionSummary
	var g errgroup.Group
	var as ActionSummary
	g.Go(func() error { return s.actionSum(appID, action, now.Add(-1*time.Hour), &as.Hour) })
	g.Go(func() error { return s.actionSum(appID, action, now.Add(-1*time.Hour*24), &as.Day) })
	g.Go(func() error { return s.actionSum(appID, action, now.Add(-1*time.Hour*24*7), &as.Week) })
	g.Go(func() error { return s.actionSum(appID, action, now.Add(-1*time.Hour*24*7*30), &as.Month) })
	g.Go(func() error { return s.actionSum(appID, action, now.Add(-1*time.Hour*24*7*365), &as.Year) })
	if err := g.Wait(); err != nil {
		return ErrCountSumFailure
	}

	// Return an Status OK
	l.Debug("Returning successful GetActionCount response")
	return c.JSON(http.StatusOK, as)
}

// incrementAction will attempt to increment the count value
// for an existing Action record for the day. If one doesn't exist
// a new one will be created with with a count of 1
func (s *Service) incrementAction(appID, action string, count int) error {
	// Get the current day and use it as a timestamp
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.Local)

	// Generate a key and execute the increment query
	key := generateKey(appID, action, today)
	return s.db.Exec(`INSERT INTO actions(id, app_id, action, count, timestamp) VALUES(?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE count = count + ?`,
		key, appID, action, count, today, count).Error
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

// actionSum will attempt to retrieve all daily actions and
// SUM them all to retrieve the total number of actions
func (s *Service) actionSum(appID, action string, startTime time.Time, out *int64) error {
	var res SumResult
	if err := s.db.Model(&Action{}).
		Select("sum(count) as total").
		Where("app_id = ?", appID).
		Where("action = ?", action).
		Where("timestamp > ?", startTime).
		Scan(&res).Error; err != nil {
		return err
	}

	// Set the count value of out passed
	*out = res.Total
	return nil
}
