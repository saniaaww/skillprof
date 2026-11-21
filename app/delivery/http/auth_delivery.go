package httpdelivery

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/saniaaww/skillprof/app/model"
	"github.com/saniaaww/skillprof/app/service"
)

type AuthDelivery struct {
	authService service.AuthService
}

func NewAuthDelivery(authService service.AuthService) *AuthDelivery {
	return &AuthDelivery{
		authService: authService,
	}
}

// Register handles user registration.
func (h *AuthDelivery) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	user, err := h.authService.Register(c.UserContext(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput),
			errors.Is(err, service.ErrWeakPassword):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})

		case errors.Is(err, service.ErrEmailAlreadyExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"message": "Email already registered",
			})

		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to register user",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Registration successful",
		"data":    user,
	})
}

// Login handles user authentication.
func (h *AuthDelivery) Login(c *fiber.Ctx) error {
	var req model.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	result, err := h.authService.Login(c.UserContext(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})

		case errors.Is(err, service.ErrInvalidCredentials):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Invalid email or password",
			})

		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to login",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Login successful",
		"data":    result,
	})
}

// Logout revokes the refresh token.
func (h *AuthDelivery) Logout(c *fiber.Ctx) error {
	var req model.RefreshTokenRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Refresh token is required",
		})
	}

	err := h.authService.Logout(c.UserContext(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Failed to logout",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Logout successful",
	})
}

// RefreshToken generates a new access and refresh token.
func (h *AuthDelivery) RefreshToken(c *fiber.Ctx) error {
	var req model.RefreshTokenRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Refresh token is required",
		})
	}

	result, err := h.authService.RefreshToken(c.UserContext(), req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid or expired refresh token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Token refreshed successfully",
		"data":    result,
	})
}

// Me returns the authenticated user's profile.
func (h *AuthDelivery) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(int64)
	if !ok || userID <= 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}

	user, err := h.authService.Me(c.UserContext(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "User profile retrieved successfully",
		"data":    user,
	})
}

// VerifyToken validates the access token from Authorization header.
func (h *AuthDelivery) VerifyToken(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Bearer token is required",
		})
	}

	result, err := h.authService.VerifyToken(c.UserContext(), parts[1])
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid or expired token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Token verification successful",
		"data":    result,
	})
}
