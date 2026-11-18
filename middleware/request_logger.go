package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func RequestLogger(logger zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)

		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = "not-provided"
		}

		event := logger.Info()

		if c.Response().StatusCode() >= 500 {
			event = logger.Error()
		} else if c.Response().StatusCode() >= 400 {
			event = logger.Warn()
		}

		event.
			Str("request_id", requestID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Msg("HTTP request")

		return err
	}
}
