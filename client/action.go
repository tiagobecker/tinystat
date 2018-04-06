package client

import (
	"fmt"

	"github.com/sdwolfe32/tinystat/models"
)

// CreateAction increments the action passed in our clients actions
// It will later on submit all actions to the Tinystat API
func (c *Client) CreateAction(action string) error {
	return c.CreateActions(action, 1)
}

// CreateActions increments the action passed in our clients actions
func (c *Client) CreateActions(action string, count int64) error {
	// Check for missing credentials on client
	if c.appID == "" || c.token == "" {
		return ErrMissingCredentials
	}

	// Store the actions
	c.Lock()
	defer c.Unlock()
	c.actions[action] = c.actions[action] + count
	return nil
}

// ActionSummary retrieves the summary of actions for the passed action
// name
func (c *Client) ActionSummary(action string) (*models.ActionSummary, error) {
	// Check for missing credentials on client
	if c.appID == "" {
		return nil, ErrMissingCredentials
	}

	// Execute the request and return the decoded count
	var summary models.ActionSummary
	path := fmt.Sprintf(actionSummaryGetPath, c.appID, action)
	return &summary, c.get(path, &summary)
}

// ActionCount retrieves the count of actions for the
// passed action name and duration
func (c *Client) ActionCount(action, duration string) (int64, error) {
	// Check for missing credentials on client
	if c.appID == "" {
		return 0, ErrMissingCredentials
	}

	// Execute the request and return the decoded count
	var count int64
	path := fmt.Sprintf(actionGetPath, c.appID, action, duration)
	return count, c.get(path, &count)
}
