package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	// baseURL is the baseURL of Tinystat
	baseURL = "https://tinystat.io"
	// timeout is the default timeout that will be set on the http.Client
	timeout = time.Duration(time.Second * 5)
)

var (
	// ErrNonOKResponse is thrown when we fail to receive a 200
	// from the Tinystat API
	ErrNonOKResponse = errors.New("Non 200 status code received")
)

// Client contains all dependencies needed to communicate
// with the Tinystat API
type Client struct {
	client       *http.Client
	appID, token string
}

// New generates a new Tinystat client
func New(appID, token string) *Client {
	return &Client{
		client: &http.Client{Timeout: timeout},
		appID:  appID,
		token:  token,
	}
}

// CountAction creates a new action on the passed metric
// It is recommended that this function be called as a goroutine
// to prevent blocking the main thread
func (c *Client) CountAction(action string) error {
	// Create the request URL
	url := fmt.Sprintf("%s/app/%s/action/%s/create", baseURL, c.appID, action)

	// Generate the request using the new URL
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// Execute the request and decode the response
	return c.do(req, nil)
}

// GetActionCount retrieves the count of actions for the
// passed metric name and duration
func (c *Client) GetActionCount(action, duration string) (int64, error) {
	// Create the request URL
	url := fmt.Sprintf("%s/app/%s/action/%s/count/%s", baseURL, c.appID, action, duration)

	// Generate the request using the new URL
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	// Execute the request and decode the response
	var count int64
	return count, c.do(req, &count)
}

// do executes the passed request and decodes the response into
// the out interface
func (c *Client) do(req *http.Request, out interface{}) error {
	// Perform the request
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check the status code of the response
	if res.StatusCode != http.StatusOK {
		return ErrNonOKResponse
	}

	// Decode the successful response if an out
	// interface is passed
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}
