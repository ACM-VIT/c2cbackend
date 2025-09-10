package codecontroller

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"c2cbackend/initializer"
	"c2cbackend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedCodesFromFile(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if user.Role != models.RoleAdmin {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}

	formFile, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing file field 'file'")
	}

	f, err := formFile.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "unable to open uploaded file")
	}
	defer f.Close()

	db := initializer.Database.Db

	reader := bufio.NewReader(f)
	seen := make(map[string]struct{})
	var codes []string
	var skippedEmpty, skippedInvalid, skippedDupInFile int

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fiber.NewError(fiber.StatusBadRequest, "failed reading uploaded file")
		}
		line = strings.TrimSpace(line)

		if line == "" {
			skippedEmpty++
		} else {
			code := strings.TrimSpace(line)
			code = strings.ToUpper(code)

			if _, dup := seen[code]; dup {
				skippedDupInFile++
			} else {
				seen[code] = struct{}{}
				codes = append(codes, code)
			}
		}

		if errors.Is(readErr, io.EOF) {
			break
		}
	}

	if len(codes) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":          "no valid codes to insert",
			"inserted":         0,
			"already_present":  0,
			"skipped_empty":    skippedEmpty,
			"skipped_invalid":  skippedInvalid,
			"skipped_dup_file": skippedDupInFile,
		})
	}

	type resultCounters struct {
		inserted       int
		alreadyPresent int
	}
	rc := resultCounters{}

	err = db.Transaction(func(tx *gorm.DB) error {
		for _, code := range codes {
			var existing models.SponCode
			if err := tx.Where("code = ?", code).First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {

					row := models.SponCode{
						ID:   uuid.New(),
						Code: code,
					}

					if err := tx.Create(&row).Error; err != nil {
						return fmt.Errorf("failed inserting code %s: %w", code, err)
					}
					rc.inserted++
					continue
				}
				return fmt.Errorf("failed checking code %s: %w", code, err)
			}
			rc.alreadyPresent++
		}
		return nil
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "seeding completed",
		"inserted":         rc.inserted,
		"already_present":  rc.alreadyPresent,
		"skipped_empty":    skippedEmpty,
		"skipped_invalid":  skippedInvalid,
		"skipped_dup_file": skippedDupInFile,
	})
}

const MAX_APPROVED_CODES_PER_TEAM = 5

func AssignCode(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if user.Role != models.RoleAdmin {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if strings.TrimSpace(body.Code) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	db := initializer.Database.Db
	var sc models.SponCode
	code := strings.ToUpper(strings.TrimSpace(body.Code))
	if err := db.Where("code = ?", code).First(&sc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "code not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if sc.TeamID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "code not requested by any team")
	}

	var approvedCount int64
	db.Model(&models.SponCode{}).
		Where("team_id = ? AND status = ?", *sc.TeamID, models.StatusApproved).
		Count(&approvedCount)
	if approvedCount >= MAX_APPROVED_CODES_PER_TEAM {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("max approved codes (%d) reached for team", MAX_APPROVED_CODES_PER_TEAM))
	}

	sc.Status = models.StatusApproved
	if err := db.Save(&sc).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    sc.Code,
		"team_id": sc.TeamID,
		"status":  sc.Status,
	})
}

// RequestCode handles user‐triggered code requests (status stays "pending").
func RequestCode(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if user.TeamID == nil || *user.TeamID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "user is not part of any team")
	}
	teamID := *user.TeamID

	db := initializer.Database.Db

	var teamCodeCount int64
	db.Model(&models.SponCode{}).
		Where("team_id = ? AND (status = ? OR status = ?)", teamID, models.StatusPending, models.StatusApproved).
		Count(&teamCodeCount)
	if teamCodeCount >= MAX_APPROVED_CODES_PER_TEAM {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("max codes (%d) reached for team", MAX_APPROVED_CODES_PER_TEAM))
	}

	var available models.SponCode
	if err := db.Where("team_id IS NULL").First(&available).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "no available codes")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	available.TeamID = &teamID
	available.Status = models.StatusPending
	available.RequestedAt = time.Now().UTC()

	if err := db.Save(&available).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":         available.Code,
		"team_id":      available.TeamID,
		"requested_at": available.RequestedAt,
		"status":       available.Status,
	})
}

func GetTeamCodes(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if user.TeamID == nil || *user.TeamID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "user is not part of any team")
	}

	db := initializer.Database.Db
	var codes []models.SponCode

	if err := db.Where("team_id = ?", *user.TeamID).Find(&codes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"codes": codes,
	})
}