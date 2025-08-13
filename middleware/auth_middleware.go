package middleware

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func FirebaseAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid Authorization header format",
			})
		}
		idToken := parts[1]

		authClient, err := initializer.FirebaseApp.Auth(context.Background())
		if err != nil {
			log.Println(err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get Firebase Auth client",
			})
		}

		decodedToken, err := authClient.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		if _, ok := decodedToken.Claims["email"]; !ok {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Email not present in token claims",
			})
		}

		email := decodedToken.Claims["email"].(string)
		var user models.User
		if err := initializer.Database.Db.Where("email = ?", email).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &fiber.Error{Code: 404, Message: "User not found"}
			}
			return &fiber.Error{Code: 500, Message: "Failed to retrieve user"}
		}

		c.Locals("user", user)
		c.Locals("uid", decodedToken.UID)
		c.Locals("claims", decodedToken.Claims)

		return c.Next()
	}
}

func FirebaseClaims() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
			return c.Next()
		}
		idToken := parts[1]

		authClient, err := initializer.FirebaseApp.Auth(context.Background())
		if err != nil {
			// Infra issue should not block public routes
			log.Println("FirebaseClaims auth client error:", err)
			return c.Next()
		}

		token, err := authClient.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			return c.Next()
		}

		c.Locals("uid", token.UID)
		c.Locals("claims", token.Claims)

		return c.Next()
	}
}
