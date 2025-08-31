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
		PPTURL      string     `json:"ppt_url"`
		Description *string    `json:"description,omitempty"`
		RoundID     uuid.UUID  `json:"round_id"`
		GithubURL   *string    `json:"github_url,omitempty"`
		FigmaURL    *string    `json:"figma_url,omitempty"`
		Other       *string    `json:"other,omitempty"`
		TrackID     *uuid.UUID `json:"track_id,omitempty"`
	}

	var input submissionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		s := strings.TrimSpace(*p)
		return &s
	}
	input.PPTURL = strings.TrimSpace(input.PPTURL)
	input.Description = trimPtr(input.Description)
	input.GithubURL = trimPtr(input.GithubURL)
	input.FigmaURL = trimPtr(input.FigmaURL)
	input.Other = trimPtr(input.Other)

	if input.RoundID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "round_id is required"})
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

	var memberCount int64
	minTeamSize, err := strconv.ParseInt(os.Getenv("TEAM_MIN_SIZE"), 10, 64)
	if err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid team min size"})
	}
	if err := tx.Model(&models.User{}).
		Where("team_id = ?", team.ID).
		Count(&memberCount).Error; err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count team members"})
	}
	if memberCount < minTeamSize {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Team must have at least %d members to submit", minTeamSize)})
	}

	var round models.Round
	if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
		Where("id = ?", input.RoundID).
		First(&round).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Round not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load round"})
	}

	// Validate PPT presence only when required by flag
	if round.PPTFlag && input.PPTURL == "" {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ppt_url is required for this round"})
	}

	var existing models.Submission
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("team_id = ? AND round_id = ?", team.ID, input.RoundID).
		First(&existing).Error
	if err == nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Submission already exists for this round"})
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check existing submission"})
	}

	teamUpdates := map[string]interface{}{}
	if input.TrackID != nil && *input.TrackID != uuid.Nil {
		teamUpdates["track_id"] = *input.TrackID
	}
	if input.GithubURL != nil {
		teamUpdates["github_url"] = *input.GithubURL
	}
	if input.FigmaURL != nil {
		teamUpdates["figma_url"] = *input.FigmaURL
	}
	if input.Other != nil {
		teamUpdates["other"] = *input.Other
	}
	if len(teamUpdates) > 0 {
		if err := tx.Model(&models.Team{}).Where("id = ?", team.ID).Updates(teamUpdates).Error; err != nil {
			_ = tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update team info"})
		}
	}

	submission := models.Submission{
		PPTURL:      input.PPTURL,
		Description: input.Description,
		RoundID:     input.RoundID,
		TeamID:      &team.ID,
	}
	if err := tx.Create(&submission).Error; err != nil {
		_ = tx.Rollback()
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "unique") || strings.Contains(low, "duplicate") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Submission already exists for this round"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create submission"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize submission"})
	}

	if err := db.Preload("Users").Preload("Track").
		First(&team, "id = ?", team.ID).Error; err != nil {
		return c.JSON(fiber.Map{
			"message":    "Submission created",
			"team":       fiber.Map{"id": team.ID},
			"submission": fiber.Map{"id": submission.ID},
		})
	}
	_ = db.Preload("Team").First(&submission, "id = ?", submission.ID).Error

	return c.JSON(fiber.Map{
		"message":    "Submission created",
		"team":       team,
		"submission": submission,
	})
}

func JoinTeamByCode(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var body models.TeamJoinSchema
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	teamCode := strings.ToUpper(strings.TrimSpace(body.Code))
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
