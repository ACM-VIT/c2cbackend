package usercontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SignUp(c *fiber.Ctx) error {
	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing auth claims",
		})
	}

	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid auth claims",
		})
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	if picture == "" {
		if v, ok := claims["profile_picture"].(string); ok {
			picture = v
		} else if v, ok := claims["profilePicture"].(string); ok {
			picture = v
		}
	}

	if email == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Email not present in token claims",
		})
	}

	type body struct {
		ContactNumber string          `json:"contact_number"`
		Gender        string          `json:"gender"`
		RegNo         string          `json:"reg_no"`
		Role          models.UserRole `json:"role"`
	}
	var req body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}
	role := req.Role
	if !models.IsValidRole(role) {
		role = models.RoleParticipant
	}

	var existing models.User
	err := initializer.Database.Db.Preload("Team").Where("email = ?", email).First(&existing).Error
	if err == nil {
		toSave := false
		if existing.Name == "" && name != "" {
			existing.Name = name
			toSave = true
		}
		if existing.ProfilePictureURL == "" && picture != "" {
			existing.ProfilePictureURL = picture
			toSave = true
		}
		if toSave {
			_ = initializer.Database.Db.Save(&existing).Error
		}
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"message": "User already exists",
			"user":    existing,
		})
	}
	if err != gorm.ErrRecordNotFound {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing user",
		})
	}

	user := models.User{
		Name:              name,
		Email:             email,
		ProfilePictureURL: picture,
		ContactNumber:     req.ContactNumber,
		Gender:            req.Gender,
		RegNo:             req.RegNo,
		Role:              role,
	}

	if _, vErr := govalidator.ValidateStruct(user); vErr != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": vErr.Error(),
		})
	}

	// Create
	if err := initializer.Database.Db.Create(&user).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"user":    user,
	})
}

func SignIn(c *fiber.Ctx) error {
	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing auth claims",
		})
	}

	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid auth claims",
		})
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Email not present in token claims",
		})
	}

	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	if picture == "" {
		if v, ok := claims["profile_picture"].(string); ok {
			picture = v
		} else if v, ok := claims["profilePicture"].(string); ok {
			picture = v
		}
	}

	var user models.User
	if err := initializer.Database.Db.Preload("Team").Where("email = ?", email).First(&user).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error":   "User not registered",
			"message": "Please complete sign-up first.",
		})
	}

	updated := false
	if user.Name == "" && name != "" {
		user.Name = name
		updated = true
	}
	if user.ProfilePictureURL == "" && picture != "" {
		user.ProfilePictureURL = picture
		updated = true
	}
	if updated {
		_ = initializer.Database.Db.Save(&user).Error
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Signed in",
		"user":    user,
	})
}
