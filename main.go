package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	httpdelivery "github.com/saniaaww/skillprof/app/delivery/http"
	"github.com/saniaaww/skillprof/app/repository"
	"github.com/saniaaww/skillprof/app/service"
	"github.com/saniaaww/skillprof/config"
	"github.com/saniaaww/skillprof/database"
	"github.com/saniaaww/skillprof/middleware"
	"github.com/saniaaww/skillprof/routes"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Println("Warning: .env file not found, using default configuration")
	}

	cfg := config.LoadAppConfig()

	logger := config.InitLogger()

	dbPool, err := database.ConnectPostgres(cfg)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed to connect to PostgreSQL")
	}
	defer dbPool.Close()

	logger.Info().Msg("PostgreSQL connected successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool)
	sessionRepo := repository.NewAuthSessionRepository(dbPool)

	// Initialize service
	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		cfg,
	)

	// Initialize delivery
	authDelivery := httpdelivery.NewAuthDelivery(authService)

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	app.Use(middleware.RequestLogger(logger))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":   "success",
			"message":  "SkillProf API is running",
			"database": "connected",
		})
	})

	app.Get("/health/database", func(c *fiber.Ctx) error {
		if err := dbPool.Ping(c.UserContext()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":   "error",
				"database": "disconnected",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":   "success",
			"database": "connected",
		})
	})

	// Register authentication routes
	routes.RegisterAuthRoutes(app, authDelivery, cfg)

	address := fmt.Sprintf(":%d", cfg.AppPort)

	logger.Info().
		Str("app", cfg.AppName).
		Int("port", cfg.AppPort).
		Msg("SkillProf server starting")

	if err := app.Listen(address); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
