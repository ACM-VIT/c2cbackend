package usercontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const internalCollegeName = "VIT University"

func SignUp(c *fiber.Ctx) error {
	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Missing auth claims"})
	}

	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid auth claims"})
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
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Email not present in token claims"})
	}

	type body struct {
		ContactNumber string          `json:"contact_number"`
		Gender        string          `json:"gender"`
		RegNo         string          `json:"reg_no"`
		Role          models.UserRole `json:"role"`
		Internal      bool            `json:"internal"`
		CollegeName   string          `json:"college_name"`
	}
	var req body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if !govalidator.Matches(req.ContactNumber, "^[6-9][0-9]{9}$") {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid mobile number. Must be 10 digits and start with 6-9."})
	}

	if req.Internal && req.RegNo == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Registration number is required for internal users",
		})
	}

	if !req.Internal && req.CollegeName == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "College name is required for external users",
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
		if req.Internal {
			existing.Internal = true
			existing.CollegeName = internalCollegeName
			toSave = true
		} else {
			if req.CollegeName != "" && existing.CollegeName == "" {
				existing.CollegeName = req.CollegeName
				toSave = true
			}
		}

		if existing.Role == "" && role != "" {
			existing.Role = role
			toSave = true
		}
		if existing.ContactNumber == "" && req.ContactNumber != "" {
			existing.ContactNumber = req.ContactNumber
			toSave = true
		}
		if existing.Gender == "" && req.Gender != "" {
			existing.Gender = req.Gender
			toSave = true
		}
		if existing.RegNo == "" && req.RegNo != "" {
			existing.RegNo = req.RegNo
			toSave = true
		}

		if toSave {
			if err := initializer.Database.Db.Save(&existing).Error; err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update existing user"})
			}
		}
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"message": "User already exists",
			"user":    existing,
		})
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check existing user"})
	}

	college := req.CollegeName
	if req.Internal {
		college = internalCollegeName
	}

	user := models.User{
		Name:              name,
		Email:             email,
		ProfilePictureURL: picture,
		ContactNumber:     req.ContactNumber,
		Gender:            req.Gender,
		RegNo:             req.RegNo,
		Internal:          req.Internal,
		CollegeName:       college,
		Role:              role,
	}

	if _, vErr := govalidator.ValidateStruct(user); vErr != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": vErr.Error()})
	}

	if err := initializer.Database.Db.Create(&user).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"user":    user,
	})
}

func SignIn(c *fiber.Ctx) error {
	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Missing auth claims"})
	}

	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid auth claims"})
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Email not present in token claims"})
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

	if user.Internal && user.CollegeName != internalCollegeName {
		user.CollegeName = internalCollegeName
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

func Dashboard(c *fiber.Ctx) error {
	if u := c.Locals("user"); u != nil {
		switch v := u.(type) {
		case models.User:
			// Reload with Team and Users to ensure members are populated.
			var user models.User
			if err := initializer.Database.Db.Preload("Team.Users").
				Where("email = ?", v.Email).
				First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
				}
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
			}
			return c.Status(http.StatusOK).JSON(fiber.Map{
				"user":         user,
				"team_members": user.Team.Users,
			})
		case *models.User:
			if v == nil {
			} else {
				var user models.User
				if err := initializer.Database.Db.Preload("Team.Users").
					Where("email = ?", v.Email).
					First(&user).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
					}
					return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
				}
				return c.Status(http.StatusOK).JSON(fiber.Map{
					"user":         user,
					"team_members": user.Team.Users,
				})
			}
		}
	}

	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Missing auth claims"})
	}
	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid auth claims"})
	}
	email, _ := claims["email"].(string)
	if email == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Email not present in token claims"})
	}

	var user models.User
	if err := initializer.Database.Db.Preload("Team.Users").
		Where("email = ?", email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"user":         user,
		"team_members": user.Team.Users,
	})
}

func GetUser(c *fiber.Ctx) error {
	if u := c.Locals("user"); u != nil {
		if user, ok := u.(models.User); ok {
			return c.Status(http.StatusOK).JSON(fiber.Map{"user": user})
		}
		if userPtr, ok := u.(*models.User); ok && userPtr != nil {
			return c.Status(http.StatusOK).JSON(fiber.Map{"user": userPtr})
		}
	}

	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Missing auth claims"})
	}
	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid auth claims"})
	}
	email, _ := claims["email"].(string)
	if email == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Email not present in token claims"})
	}

	var user models.User
	if err := initializer.Database.Db.Preload("Team").Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"user": user})
}
