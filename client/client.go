package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// baseURL is the baseURL of Tinystat
const baseURL = "https://tinystat.io"

var (
	// DefaultClient is the client that will be used for all metrics
	// reporting
	DefaultClient = newClient(time.Second*5, time.Second*10)
	// ErrNonOKResponse is thrown when we fail to receive a 200
	// from the Tinystat API
	ErrNonOKResponse = errors.New("Non 200 status code received")
	// ErrMissingCredentials is thrown when we fail to find
	// an appID or token on the DefaultClient
	ErrMissingCredentials = errors.New("Tinystat credentials are missing")
)

// Client contains all dependencies needed to communicate
// with the Tinystat API
type Client struct {
	sync.RWMutex
	client  *http.Client
	actions map[string]int64 // action -> count
	appID   string
	token   string
}

// New generates a new Tinystat client
func newClient(timeout, sendFreq time.Duration) *Client {
	c := &Client{
		client:  &http.Client{Timeout: timeout},
		actions: make(map[string]int64),
		appID:   os.Getenv("TINYSTAT_APP_ID"),
		token:   os.Getenv("TINYSTAT_TOKEN"),
	}
	go c.sendWorker(sendFreq)
	return c
}

// sendWorker periodically sends new actions to the Tinystat API
// It is done this way to prevent overwhelming the server
func (c *Client) sendWorker(sendFreq time.Duration) {
	// Don't start worker if appID and token aren't set
	if c.appID == "" || c.token == "" {
		return
	}

	// Begin infinite send loop
	for {
		time.Sleep(sendFreq)
		c.Lock()
		// Create an action for every count
		for action, count := range c.actions {
			// Perform the request
			c.get(fmt.Sprintf("/app/%s/action/%s/create?count=%v",
				c.appID, action, count), nil)
		}
		c.actions = make(map[string]int64)
		c.Unlock()
	}
}

// CreateAction increments the action passed in our clients actions
// It will later on submit all actions to the Tinystat API
func (c *Client) createAction(action string) error {
	// Check for missing credentials on client
	if c.appID == "" || c.token == "" {
		return ErrMissingCredentials
	}

	// Store the action
	c.Lock()
	defer c.Unlock()
	c.actions[action]++
	return nil
}

// getActionCount retrieves the count of actions for the
// passed action name and duration
func (c *Client) getActionCount(action, duration string) (int64, error) {
	// Check for missing credentials on client
	if c.appID == "" || c.token == "" {
		return 0, ErrMissingCredentials
	}

	// Execute the request and return the decoded count
	var count int64
	return count, c.get(fmt.Sprintf("/app/%s/action/%s/count/%s",
		c.appID, action, duration), &count)
}

// post performs a POST request using the provided path, in body
// interface and out response interface
func (c *Client) post(path string, in, out interface{}) error {
	return c.do(http.MethodPost, path, in, out)
}

// get performs a GET request using the provided path, and out
// response interface
func (c *Client) get(path string, out interface{}) error {
	return c.do(http.MethodGet, path, nil, out)
}

// do executes the passed request and decodes the response into
// the out interface
func (c *Client) do(method, path string, in, out interface{}) error {
	// Marshal a request body if one exists
	var body io.Reader
	if in != nil {
		jsonBytes, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(jsonBytes)
	}

	// Generate the request and append auth headers
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Add("TOKEN", c.token)

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
