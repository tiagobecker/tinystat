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
func CreateAction(action string) {
	go DefaultClient.CreateAction(action)
}

// GetActionSummary retrieves an action summary using the DefaultClient
func GetActionSummary(action string) (*models.ActionSummary, error) {
	return DefaultClient.ActionSummary(action)
}

// GetActionCount retrieves action stats using the DefaultClient
func GetActionCount(action, duration string) (int64, error) {
	return DefaultClient.ActionCount(action, duration)
}
