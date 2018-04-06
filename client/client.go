package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// baseURL is the baseURL of Tinystat
const (
	baseURL              = "https://tinystat.io"
	actionPostPath       = "/app/%s/action/%s/create/%v"
	actionSummaryGetPath = "/app/%s/action/%s/count"
	actionGetPath        = "/app/%s/action/%s/count/%s"
)

var (
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

// NewClient generates a new standard Client using the passed
// timeout, sendFreq, appID and token
func NewClient(appID, token string, timeout, sendFreq time.Duration) *Client {
	// Generate a new client, apply the APP_ID and TOKEN
	c := newClient(timeout)
	c.SetAppID(appID)
	c.SetToken(token)

	// Begin the worker and return
	go c.sendWorker(sendFreq)
	return c
}

// newClient generates a new basic Tinystat Client using the
// passed http timeout and send frequency
func newClient(timeout time.Duration) *Client {
	return &Client{
		client:  &http.Client{Timeout: timeout},
		actions: make(map[string]int64),
	}
}

// SetAppID sets the AppID on a Client
func (c *Client) SetAppID(appID string) { c.appID = appID }

// SetToken sets the Token on the Client
func (c *Client) SetToken(token string) { c.token = token }

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
			go c.post(fmt.Sprintf(actionPostPath, c.appID, action, count), nil, nil)
		}
		c.actions = make(map[string]int64)
		c.Unlock()
	}
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
	// if found on the client
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Add("TOKEN", c.token)
	}

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
