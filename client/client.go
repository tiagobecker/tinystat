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

const (
	// baseURL is the baseURL of Tinystat
	baseURL = "https://tinystat.io"
	// timeout is the default timeout that will be set on the http.Client
	timeout = time.Duration(time.Second * 5)
	// sendFreq is the frequency of at which counts will be submitted
	sendFreq = time.Duration(time.Second * 10)
)

var (
	// ErrNonOKResponse is thrown when we fail to receive a 200
	// from the Tinystat API
	ErrNonOKResponse = errors.New("Non 200 status code received")
)

// Client contains all dependencies needed to communicate
// with the Tinystat API
type Client struct {
	client    *http.Client
	actionMap *ActionMap
	appID     string
	token     string
}

// ActionMap is a thread safe map which holds counts of actions
type ActionMap struct {
	sync.RWMutex
	actions map[string]int64 // action -> count
}

// New generates a new Tinystat client
func New(appID, token string) *Client {
	c := &Client{
		client:    &http.Client{Timeout: timeout},
		actionMap: &ActionMap{actions: make(map[string]int64)},
		appID:     appID,
		token:     token,
	}
	go c.sendWorker()
	return c
}

// CreateAction increments the action passed in our clients actions
// It will later on submit all actions to the Tinystat API
func (c *Client) CreateAction(action string) {
	c.actionMap.Lock()
	defer c.actionMap.Unlock()
	c.actionMap.actions[action]++
}

// GetActionCount retrieves the count of actions for the
// passed action name and duration
func (c *Client) GetActionCount(action, duration string) (int64, error) {
	// Create the request URL
	path := fmt.Sprintf("/app/%s/action/%s/count/%s", c.appID, action, duration)

	// Execute the request return the decoded response
	var count int64
	return count, c.get(path, &count)
}

// sendWorker periodically sends new actions to the Tinystat API
// It is done this way to prevent overwhelming the server
func (c *Client) sendWorker() {
	for {
		time.Sleep(sendFreq)
		c.actionMap.Lock()
		// Create an action for every count
		for action, count := range c.actionMap.actions {
			// Create the request URL
			path := fmt.Sprintf("/app/%s/action/%s/create?count=%v", c.appID, action, count)

			// Perform the request
			c.get(path, nil)
		}
		c.actionMap.actions = make(map[string]int64)
		c.actionMap.Unlock()
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
