package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
)

// rateLimit is the amount of time a requestor must wait before
// making another request
const rateLimit = time.Second * 1 // 1RPS

var (
	// ErrRateLimitExceeded is thrown when an IP exceeds the specified rate limit
	ErrRateLimitExceeded = echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded (1RPS)")
	// ErrInvalidToken is thrown when a request fails to be authenticated
	ErrInvalidToken = echo.NewHTTPError(http.StatusUnauthorized, "Failed to validate token")
)

// TokenAuth validates that the token matches the appID
// If the strictAuth value is set to true, a token MUST be valid
// If the strictAuth value is set to false, we'll refer to the users secure flag
func (s *Service) TokenAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		l := s.logger.WithField("method", "validate_token")

		// Pull the appID and token from the request
		appID := c.Param("app_id")
		var token string
		token = c.QueryParam("token")
		if token == "" {
			token = c.Request().Header.Get("TOKEN")
		}

		// Check the cache for a stored app/token and validate
		l.Debug("Checking cache for App")
		var app App
		if appIface, ok := s.cache.Get(appID); ok {
			app = *appIface.(*App)
		} else {
			// Attempt to retrieve the app from the DB if it couldn't be found in cache
			if err := s.db.Where(&App{ID: appID}).Find(&app).Error; err != nil {
				l.WithError(err).Error("Failed to retrieve App from DB")
				return ErrInvalidToken
			}
			// Cache the App for future
			l.Debug("Storing App in Cache")
			s.cache.SetDefault(appID, &app)
		}

		// If a POST request or a secure app (secure all get requests) verify token
		if c.Request().Method == http.MethodPost || app.StrictAuth {
			if app.Token != token {
				l.WithError(ErrInvalidToken).Error("Failed to validate token")
				return ErrInvalidToken
			}
		}
		// Otherwise fuck it
		return next(c)
	}
}

// RateLimit returns true if the ip passed has performed too
// many requests lately
// vars should include the IP and any other variables to make
// rate limiting unique to a path
func (s *Service) RateLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		l := s.logger.WithField("method", "rate_limit")

		// Create a vars key
		var vars []string
		vars = append(vars, c.RealIP())
		vars = append(vars, c.ParamValues()...)
		key := strings.Join(vars, "_")
		l = l.WithField("key", key)
		l.Debug("Performing rate limit check")

		// Lock the map
		s.rateMap.Lock()
		defer s.rateMap.Unlock()

		// If this IP is in the map and it's last request
		// was within the specified ratelimit timeframe
		if last, ok := s.rateMap.ipMap[key]; ok &&
			last.After(time.Now().Add(-1*rateLimit)) {
			l.WithError(ErrRateLimitExceeded).Error("Rate limit exceeded")
			return ErrRateLimitExceeded
		}

		// Set a new last request time and allow the request
		s.rateMap.ipMap[key] = time.Now()
		return next(c)
	}
}
