package teamcontroller

import (
	"c2cbackend/helpers"
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateTeam(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	if user.TeamID != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is already in a team"})
	}

	type CreateTeamInput struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	var input CreateTeamInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team name is required"})
	}

	db := initializer.Database.Db
	tx := db.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var team models.Team
	const maxRetries = 3
	var createErr error
	for i := 0; i < maxRetries; i++ {
		code := helpers.GenerateTeamCode()
		team = models.Team{
			Name:        input.Name,
			Description: input.Description,
			Code:        code,
		}
		createErr = tx.Create(&team).Error
		if createErr == nil {
			break
		}
		if !errors.Is(createErr, gorm.ErrDuplicatedKey) {
			break
		}
	}
	if createErr != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create team"})
	}

	var freshUser models.User
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", user.ID).
		First(&freshUser).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load user"})
	}
	if freshUser.TeamID != "" {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is already in a team"})
	}

	res := tx.Model(&models.User{}).
		Where("id = ? AND team_id IS NULL", freshUser.ID).
		Update("team_id", team.ID)
	if res.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign user to team"})
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Team assignment race, please retry"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize team creation"})
	}

	if err := db.Preload("Users").First(&team, "id = ?", team.ID).Error; err == nil {
		return c.JSON(fiber.Map{
			"message": "Team created successfully",
			"team":    team,
			"code":    team.Code,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Team created successfully",
		"team":    team,
		"code":    team.Code,
	})
}

func JoinTeamByCode(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var body models.TeamJoinSchema
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	teamCode := strings.TrimSpace(body.Code)
	if teamCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team code is required"})
	}

	const maxTeamSize int64 = 2
	db := initializer.Database.Db

	tx := db.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var team models.Team
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code = ?", teamCode).
		First(&team).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve team"})
	}

	var freshUser models.User
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", user.ID).
		First(&freshUser).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load user"})
	}
	if freshUser.TeamID != "" {
		tx.Rollback()
		return c.Status(400).JSON(fiber.Map{"error": "User is already in a team"})
	}

	var memberCount int64
	if err := tx.Model(&models.User{}).
		Where("team_id = ?", team.ID).
		Count(&memberCount).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count team members"})
	}
	if memberCount >= maxTeamSize {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team is full"})
	}

	// Join the team (with guard)
	res := tx.Model(&models.User{}).
		Where("id = ? AND team_id IS NULL", freshUser.ID).
		Update("team_id", team.ID)
	if res.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to join team"})
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is already in a team"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to join team"})
	}

	// Load users for the response
	if err := db.Preload("Users").First(&team, "id = ?", team.ID).Error; err == nil {
		return c.JSON(fiber.Map{
			"message": "Successfully joined team",
			"team":    team,
		})
	}

	// Fallback if preload fails
	return c.JSON(fiber.Map{
		"message": "Successfully joined team",
		"team":    team,
	})
}

func LeaveTeam(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	db := initializer.Database.Db
	tx := db.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Re-load and lock the user (state may have changed since middleware ran)
	var freshUser models.User
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", user.ID).
		First(&freshUser).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load user"})
	}

	if freshUser.TeamID == "" {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is not in a team"})
	}

	oldTeamID := freshUser.TeamID

	res := tx.Model(&models.User{}).
		Where("id = ? AND team_id = ?", freshUser.ID, oldTeamID).
		Update("team_id", gorm.Expr("NULL"))
	if res.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to leave team"})
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Team membership changed, please retry"})
	}

	if err := tx.Exec("DELETE FROM team_users WHERE user_id = ? AND team_id = ?", freshUser.ID, oldTeamID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update team membership"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize leave"})
	}

	return c.JSON(fiber.Map{
		"message": "Left team successfully",
		"user_id": freshUser.ID,
		"team_id": oldTeamID,
	})
}
