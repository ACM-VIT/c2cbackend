package dashboardcontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// todo: implement checkin flag in rounds and user checkedin field so that the portal can conditionally render check in CTA
func Dashboard(c *fiber.Ctx) error {
	// Get the authenticated user
	var ctxUser models.User
	switch v := c.Locals("user").(type) {
	case *models.User:
		if v == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		ctxUser = *v
	case models.User:
		ctxUser = v
	default:
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	db := initializer.Database.Db

	// Load user with team, teammates, and track (track may be null/zero)
	var user models.User
	q := db.Model(&models.User{}).
		Select([]string{
			"id", "created_at", "updated_at",
			"name", "email", "profile_picture_url", "contact_number",
			"gender", "reg_no", "internal", "college_name", "role", "team_id",
		}).
		Where("id = ?", ctxUser.ID).
		Preload("Team", func(db *gorm.DB) *gorm.DB {
			return db.Select([]string{
				"id", "created_at", "updated_at",
				"name", "description", "code",
				"github_url", "figma_url", "other",
				"track_id", "tech_stack",
			})
		}).
		Preload("Team.Track", func(db *gorm.DB) *gorm.DB {
			return db.Select([]string{"id", "created_at", "updated_at", "title", "description"})
		}).
		Preload("Team.Users", func(db *gorm.DB) *gorm.DB {
			return db.
				Select([]string{
					"id", "created_at", "updated_at",
					"name", "email", "profile_picture_url",
					"contact_number", "gender", "reg_no",
					"internal", "college_name", "role", "team_id",
				}).
				Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: false})
		})

	if err := q.First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}

	// "user": current user without nested team
	currentUser := user
	currentUser.Team = nil

	var teamResp interface{} = nil
	var trackResp interface{} = fiber.Map{}
	teammates := []models.User{}

	if user.Team != nil {

		teamResp = fiber.Map{
			"id":          user.Team.ID,
			"created_at":  user.Team.CreatedAt,
			"updated_at":  user.Team.UpdatedAt,
			"name":        user.Team.Name,
			"description": user.Team.Description,
			"code":        user.Team.Code,
			"github_url":  user.Team.GithubURL,
			"figma_url":   user.Team.FigmaURL,
			"other":       user.Team.Other,
			"track_id":    user.Team.TrackID,
			"tech_stack":  user.Team.TechStack,
		}

		for _, u := range user.Team.Users {
			if u.ID == user.ID {
				continue
			}
			u.Team = nil
			teammates = append(teammates, u)
		}

		// track in separate field; {} if no track id, else populated object
		if user.Team.TrackID != nil && *user.Team.TrackID != uuid.Nil && user.Team.Track.ID != uuid.Nil {
			tr := user.Team.Track
			trackResp = tr
		}
	}

	submissionResp := interface{}(nil)
	submitted := false
	if user.Team != nil {
		var sub models.Submission
		err := db.Model(&models.Submission{}).
			Preload("Round").
			Select([]string{"ppt_url", "title", "description", "round_id"}).
			Where("team_id = ?", user.Team.ID).
			Order("created_at desc").
			First(&sub).Error
		switch err {
		case nil:
			submitted = true
			submissionResp = fiber.Map{
				"ppt_url":        sub.PPTURL,
				"title":          sub.Title,
				"description":    sub.Description,
				"round_end_time": sub.Round.EndTime,
			}
		case gorm.ErrRecordNotFound:
			submitted = false
			submissionResp = fiber.Map{}
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load submission"})
		}
	}

	minTeamSize, err := strconv.ParseInt(os.Getenv("TEAM_MIN_SIZE"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to parse min team size"})
	}

	// Determine the current round for the team (highest round in round_teams)
	currentRoundResp := interface{}(nil)
	activeRoundResp := interface{}(nil)
	if user.Team != nil {
		var curr models.Round
		err := db.Model(&models.Round{}).
			Select([]string{
				"rounds.id", "rounds.created_at", "rounds.updated_at",
				"rounds.name", "rounds.round_number", "rounds.screen_flag", "rounds.ppt_flag",
				"rounds.start_time", "rounds.end_time", "rounds.description", "rounds.check_in_flag",
			}).
			Joins("JOIN round_teams rt ON rt.round_id = rounds.id").
			Where("rt.team_id = ?", user.Team.ID).
			Order("rounds.round_number DESC").
			Limit(1).
			First(&curr).Error
		switch err {
		case nil:
			currentRoundResp = fiber.Map{
				"id":            curr.ID,
				"created_at":    curr.CreatedAt,
				"updated_at":    curr.UpdatedAt,
				"name":          curr.Name,
				"round_number":  curr.RoundNumber,
				"screen_flag":   curr.ScreenFlag,
				"ppt_flag":      curr.PPTFlag,
				"start_time":    curr.StartTime,
				"end_time":      curr.EndTime,
				"description":   curr.Description,
				"check_in_flag": curr.CheckInFlag,
			}
		case gorm.ErrRecordNotFound:
			currentRoundResp = fiber.Map{}
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load current round"})
		}

		// Load all rounds the team has participated in
		var teamRounds []models.Round
		if err := db.Model(&models.Round{}).
			Select([]string{
				"rounds.id", "rounds.created_at", "rounds.updated_at",
				"rounds.name", "rounds.round_number", "rounds.screen_flag", "rounds.ppt_flag",
				"rounds.start_time", "rounds.end_time", "rounds.description", "rounds.check_in_flag",
			}).
			Joins("JOIN round_teams rt ON rt.round_id = rounds.id").
			Where("rt.team_id = ?", user.Team.ID).
			Order("rounds.round_number ASC").
			Find(&teamRounds).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load team rounds"})
		}
	}

	// Determine the currently active round based on time (global)
	now := time.Now().UTC()
	{
		var active models.Round
		err := db.Model(&models.Round{}).
			Select([]string{
				"id", "created_at", "updated_at",
				"name", "round_number", "screen_flag", "ppt_flag",
				"start_time", "end_time", "description", "check_in_flag",
			}).
			Where("start_time <= ? AND end_time >= ?", now, now).
			Order("round_number DESC").
			Limit(1).
			First(&active).Error
		switch err {
		case nil:
			activeRoundResp = fiber.Map{
				"id":            active.ID,
				"created_at":    active.CreatedAt,
				"updated_at":    active.UpdatedAt,
				"name":          active.Name,
				"round_number":  active.RoundNumber,
				"screen_flag":   active.ScreenFlag,
				"ppt_flag":      active.PPTFlag,
				"start_time":    active.StartTime,
				"end_time":      active.EndTime,
				"description":   active.Description,
				"check_in_flag": active.CheckInFlag,
			}
		case gorm.ErrRecordNotFound:
			activeRoundResp = fiber.Map{}
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load active round"})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":               currentUser,
		"team":               teamResp,
		"teammates":          teammates,
		"track":              trackResp,
		"submission":         submissionResp,
		"submitted":          submitted,
		"current_team_round": currentRoundResp,
		"active_round":       activeRoundResp,
		"minmembercount":     minTeamSize,
		"c2chappening":       os.Getenv("C2C_HAPPENING") == "true",
	})
}
