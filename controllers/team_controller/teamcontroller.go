package teamcontroller

import (
	"c2cbackend/helpers"
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateTeam(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	// User must not already be in a team
	if user.TeamID != nil {
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
			_ = tx.Rollback()
		}
	}()

	// Create team (no track handling)
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
		// If the error is due to duplicate code, retry; otherwise break
		if !errors.Is(createErr, gorm.ErrDuplicatedKey) {
			break
		}
	}
	if createErr != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create team"})
	}

	// Lock & reload user, then assign team if still free
	var freshUser models.User
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", user.ID).
		First(&freshUser).Error; err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load user"})
	}
	if freshUser.TeamID != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is already in a team"})
	}

	res := tx.Model(&models.User{}).
		Where("id = ? AND team_id IS NULL", freshUser.ID).
		Update("team_id", team.ID)
	if res.Error != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign user to team"})
	}
	if res.RowsAffected == 0 {
		_ = tx.Rollback()
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

func CreateTeamSubmission(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if user.TeamID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is not in a team"})
	}

	type submissionInput struct {
		GithubURL string    `json:"github_url"`
		FigmaURL  string    `json:"figma_url"`
		Other     string    `json:"other"`
		TrackID   uuid.UUID `json:"track_id"`
	}

	var input submissionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	input.GithubURL = strings.TrimSpace(input.GithubURL)
	input.FigmaURL = strings.TrimSpace(input.FigmaURL)
	input.Other = strings.TrimSpace(input.Other)

	// Validate required fields
	if input.GithubURL == "" || input.FigmaURL == "" || input.Other == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "All of github_url, figma_url and other are required",
		})
	}
	if input.TrackID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "track_id is required",
		})
	}

	db := initializer.Database.Db
	tx := db.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	var track models.Track
	if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
		Where("id = ?", input.TrackID).
		First(&track).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Track not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load track"})
	}

	var memberCount int64
	minTeamSize, err := strconv.ParseInt(os.Getenv("TEAM_MIN_SIZE"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid team min size"})
	}

	if err := tx.Model(&models.User{}).
		Where("team_id = ?", *user.TeamID).
		Count(&memberCount).Error; err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count team members"})
	}
	if memberCount < minTeamSize {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Team must have at least %d members to submit", minTeamSize)})
	}

	var team models.Team
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", *user.TeamID).
		First(&team).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load team"})
	}

	updates := map[string]interface{}{
		"github_url": input.GithubURL,
		"figma_url":  input.FigmaURL,
		"other":      input.Other,
		"track_id":   input.TrackID,
	}
	if err := tx.Model(&models.Team{}).
		Where("id = ?", team.ID).
		Updates(updates).Error; err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save submission"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize submission"})
	}

	// Return the updated team
	if err := db.Preload("Users").
		Preload("Track").
		First(&team, "id = ?", team.ID).Error; err != nil {
		return c.JSON(fiber.Map{
			"message": "Submission saved",
			"team":    fiber.Map{"id": team.ID},
		})
	}

	return c.JSON(fiber.Map{
		"message": "Submission saved",
		"team":    team,
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

	maxTeamSize, err := strconv.ParseInt(os.Getenv("TEAM_MAX_SIZE"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid team max size"})
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
	if freshUser.TeamID != nil {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is already in a team"})
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

	if err := db.Preload("Users").Preload("Track").First(&team, "id = ?", team.ID).Error; err == nil {
		return c.JSON(fiber.Map{
			"message": "Successfully joined team",
			"team":    team,
		})
	}

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

	var freshUser models.User
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", user.ID).
		First(&freshUser).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load user"})
	}

	if freshUser.TeamID == nil {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is not in a team"})
	}

	oldTeamID := *freshUser.TeamID

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

	del := tx.Exec(`
  DELETE FROM teams
  WHERE id = ?
    AND NOT EXISTS (SELECT 1 FROM users WHERE team_id = ?)
`, oldTeamID, oldTeamID)

	if del.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete team"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize leave"})
	}

	return c.JSON(fiber.Map{
		"message":     "Left team successfully",
		"user_id":     freshUser.ID,
		"team_id":     oldTeamID,
		"teamDeleted": del.RowsAffected > 0,
	})
}
