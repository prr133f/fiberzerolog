package fiberzerolog

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// New creates a new middleware handler
func New(config ...Config) fiber.Handler {
	// Set default config
	cfg := configDefault(config...)

	// put ignore uri into a map for faster match
	skipURIs := make(map[string]struct{}, len(cfg.SkipURIs))
	for _, uri := range cfg.SkipURIs {
		skipURIs[uri] = struct{}{}
	}

	// Return new handler
	return func(c *fiber.Ctx) error {
		// Don't execute middleware if Next returns true
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// skip uri
		if _, ok := skipURIs[c.Path()]; ok {
			return c.Next()
		}

		start := time.Now()

		errorTraceID := uuid.New()
		c.Set("errorTraceID", errorTraceID.String())

		// Handle request, store err for logging
		chainErr := c.Next()
		if chainErr != nil {
			// Manually call error handler
			if err := c.App().ErrorHandler(c, chainErr); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		latency := time.Since(start)

		status := c.Response().StatusCode()

		index := 0
		switch {
		case status >= 500:
			// error index is zero
		case status >= 400:
			index = 1
		default:
			index = 2
		}

		levelIndex := index
		if levelIndex >= len(cfg.Levels) {
			levelIndex = len(cfg.Levels) - 1
		}
		level := cfg.Levels[levelIndex]

		// no log
		if level == zerolog.NoLevel || level == zerolog.Disabled {
			return nil
		}

		messageIndex := index
		if messageIndex >= len(cfg.Messages) {
			messageIndex = len(cfg.Messages) - 1
		}
		message := cfg.Messages[messageIndex]

		logger := cfg.logger(c, latency, chainErr)
		ctx := c.UserContext()

		switch level {
		case zerolog.DebugLevel:
			logger.Debug().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		case zerolog.InfoLevel:
			logger.Info().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		case zerolog.WarnLevel:
			logger.Warn().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		case zerolog.ErrorLevel:
			logger.Error().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		case zerolog.FatalLevel:
			logger.Fatal().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		case zerolog.PanicLevel:
			logger.Panic().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		case zerolog.TraceLevel:
			logger.Trace().Ctx(ctx).Any("errorTraceId", errorTraceID).Msg(message)
		}

		return nil
	}
}
