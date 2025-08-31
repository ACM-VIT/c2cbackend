package teamcontroller

import (
	"c2cbackend/helpers"
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateTeam(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	if os.Getenv("TEAM_LOCK") == "TRUE" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team creation is currently locked"})
	}

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

	{
		var earliestRound models.Round
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("start_time IS NOT NULL").
			Order("start_time ASC").
			Limit(1).
			First(&earliestRound).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			_ = tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find earliest round"})
		}
		if err == nil {
			// insert into join table within the same transaction
			if exec := tx.Exec("INSERT INTO round_teams (round_id, team_id) VALUES (?, ?)", earliestRound.ID, team.ID); exec.Error != nil {
				_ = tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to associate team with earliest round"})
			}
		}
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

	// track_id is MANDATORY now
	type submissionInput struct {
		PPTURL      string     `json:"ppt_url"`
		Description *string    `json:"description,omitempty"`
		GithubURL   *string    `json:"github_url,omitempty"`
		FigmaURL    *string    `json:"figma_url,omitempty"`
		Other       *string    `json:"other,omitempty"`
		TrackID     *uuid.UUID `json:"track_id"` // required
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

	// Load team with active round(s) now
	now := time.Now()
	var team models.Team
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Rounds", "start_time <= ? AND end_time >= ?", now, now).
		Where("id = ?", *user.TeamID).
		First(&team).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load team"})
	}

	if len(team.Rounds) == 0 {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No active round for this team at the current time"})
	}
	currentRound := team.Rounds[0]

	if input.TrackID == nil || *input.TrackID == uuid.Nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "track_id is required"})
	}

	if currentRound.PPTFlag && input.PPTURL == "" {
		_ = tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ppt_url is required for this round"})
	}

	// Team size check
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

	// Ensure no prior submission for this (team, round)
	var existing models.Submission
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("team_id = ? AND round_id = ?", team.ID, currentRound.ID).
		First(&existing).Error
	if err == nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Submission already exists for this round"})
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check existing submission"})
	}

	// Update Team fields (track_id is mandatory → always update)
	if err := tx.Model(&models.Team{}).
		Where("id = ?", team.ID).
		Update("track_id", *input.TrackID).Error; err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update team track"})
	}

	teamUpdates := map[string]interface{}{}
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

	// Create submission
	submission := models.Submission{
		PPTURL:      input.PPTURL,
		Description: input.Description,
		RoundID:     currentRound.ID,
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

	// Auto-promote: if ScreenFlag is true AND team has any external participant, move to NEXT round
	promoted := false
	if currentRound.ScreenFlag {
		var externalCount int64
		if err := tx.Model(&models.User{}).
			Where("team_id = ? AND internal = ?", team.ID, false).
			Count(&externalCount).Error; err != nil {
			_ = tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check team composition"})
		}

		if externalCount > 0 {
			log.Println("Auto-promoting team", team.ID, "from round", currentRound.ID)
			var nextRound models.Round
			if err := tx.Where("round_number > ?", currentRound.RoundNumber).
				Order("round_number ASC").
				First(&nextRound).Error; err == nil {

				if err := tx.Exec(`
					INSERT INTO round_teams (round_id, team_id)
					VALUES (?, ?)
					ON CONFLICT (round_id, team_id) DO NOTHING
				`, nextRound.ID, team.ID).Error; err != nil {
					_ = tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to promote team to next round"})
				}

				// 2) remove current link (so the team is actually moved)
				if err := tx.Exec(`
					DELETE FROM round_teams
					WHERE round_id = ? AND team_id = ?
				`, currentRound.ID, team.ID).Error; err != nil {
					_ = tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove team from current round"})
				}

				promoted = true
			}
			// if no next round exists, skip silently
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize submission"})
	}

	// Reload with associations (include Rounds so caller can see current membership)
	if err := db.Preload("Users").
		Preload("Track").
		Preload("Rounds").
		First(&team, "id = ?", team.ID).Error; err != nil {
		return c.JSON(fiber.Map{
			"message":    "Submission created",
			"promoted":   promoted,
			"team":       fiber.Map{"id": team.ID},
			"submission": fiber.Map{"id": submission.ID},
		})
	}
	_ = db.Preload("Team").First(&submission, "id = ?", submission.ID).Error

	return c.JSON(fiber.Map{
		"message":    "Submission created",
		"promoted":   promoted,
		"team":       team,
		"submission": submission,
	})
}

func JoinTeamByCode(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	if os.Getenv("TEAM_LOCK") == "TRUE" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team joining is currently locked"})
	}

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

	if os.Getenv("TEAM_LOCK") == "TRUE" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Team leaving is currently locked"})
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

func GetTeamSubmission(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if user.TeamID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User is not in a team"})
	}

	db := initializer.Database.Db

	now := time.Now()
	var team models.Team
	if err := db.
		Preload("Rounds", "start_time <= ? AND end_time >= ?", now, now).
		First(&team, "id = ?", *user.TeamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load team"})
	}

	if len(team.Rounds) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No active round for this team"})
	}

	currentRound := team.Rounds[0]

	var submission models.Submission
	if err := db.
		Preload("Team").
		Where("team_id = ? AND round_id = ?", team.ID, currentRound.ID).
		First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Submission not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve submission"})
	}

	return c.JSON(fiber.Map{
		"message":    "Submission retrieved",
		"submission": submission,
	})
}
