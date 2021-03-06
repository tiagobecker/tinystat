package client

import (
	"os"
	"time"

	"github.com/sdwolfe32/tinystat/models"
)

// DefaultClient is the client that will be used for all metrics
// reporting
var DefaultClient = NewClient(os.Getenv("TINYSTAT_APP_ID"), os.Getenv("TINYSTAT_TOKEN"),
	"https://tinystat.io", 1, time.Second*5, time.Second*10)

// CreateAction creates a new action using the DefaultClient
func CreateAction(action string) error {
	return DefaultClient.CreateAction(action)
}

// ActionSummary retrieves an action summary using the DefaultClient
func ActionSummary(action string) (*models.ActionSummary, error) {
	return DefaultClient.ActionSummary(action)
}

// ActionCount retrieves action stats using the DefaultClient
func ActionCount(action, duration string) (int64, error) {
	return DefaultClient.ActionCount(action, duration)
}
