package api

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/sdwolfe32/tinystat/client"
	"golang.org/x/sync/errgroup"
)

// ErrStatsRetrievalFailure is thrown when we fail to retrieve stats
var ErrStatsRetrievalFailure = echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve Stats")

// Stats is any action that can be stored with a timestamp
type Stats struct {
	Apps                int64 `json:"apps"`
	ActionsRecorded     int64 `json:"actionsRecorded"`
	CountsCalculated    int64 `json:"countsCalculated"`
	SummariesCalculated int64 `json:"summariesCalculated"`
}

// Stats returns the overall stats for Tinystat
// Endpoint: /stats
func (s *Service) Stats(c echo.Context) error {
	l := s.logger.WithField("method", "stats")
	l.Debug("Received new Stats request")

	// Store the new action in the database
	l.Debug("Retrieving Tinystat stats")
	var g errgroup.Group
	var stats Stats
	g.Go(func() error { return s.db.Model(&App{}).Count(&stats.Apps).Error })
	g.Go(func() error {
		return s.actionSum(&stats.ActionsRecorded, "app_id = ?", s.appID, "action = ?", "create-action")
	})
	g.Go(func() error {
		return s.actionSum(&stats.CountsCalculated, "app_id = ?", s.appID, "action = ?", "action-count")
	})
	g.Go(func() error {
		return s.actionSum(&stats.SummariesCalculated, "app_id = ?", s.appID, "action = ?", "action-summary")
	})
	if err := g.Wait(); err != nil {
		l.WithError(err).Error("Failed to retrieve overall stats")
		return ErrStatsRetrievalFailure
	}

	// Report the successful get-stats to ourselves
	go client.CreateAction("stats")

	// Return a Status OK
	l.Debug("Returning successful Stats response")
	return c.JSON(http.StatusOK, stats)
}
