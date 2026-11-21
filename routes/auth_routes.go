package routes

import (
	"github.com/gofiber/fiber/v2"
	httpdelivery "github.com/saniaaww/skillprof/app/delivery/http"
	"github.com/saniaaww/skillprof/config"
	"github.com/saniaaww/skillprof/middleware"
)

func RegisterAuthRoutes(
	api fiber.Router,
	authDelivery *httpdelivery.AuthDelivery,
	cfg config.AppConfig,
) {
	auth := api.Group("/auth")

	auth.Post("/register", authDelivery.Register)
	auth.Post("/login", authDelivery.Login)
	auth.Post("/logout", authDelivery.Logout)
	auth.Post("/refresh", authDelivery.RefreshToken)
	auth.Get("/me", middleware.AuthMiddleware(cfg), authDelivery.Me)
	auth.Post("/verify-token", authDelivery.VerifyToken)
}
