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
                        Code: &code,
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

    // If team already has an approved code, return it (only approved codes are returned)
    var approved models.SponCode
    if err := db.Where("team_id = ? AND status = ?", teamID, models.StatusApproved).
        Order("requested_at ASC").First(&approved).Error; err == nil {
        return c.Status(fiber.StatusOK).JSON(fiber.Map{
            "code":         approved.Code,
            "team_id":      approved.TeamID,
            "requested_at": approved.RequestedAt,
            "status":       approved.Status,
        })
    } else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }


    // Count pending requests
    var pendingCount int64
    if err := db.Model(&models.SponCode{}).
        Where("team_id = ? AND status = ?", teamID, models.StatusPending).
        Count(&pendingCount).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }

    // If no pending and no approved, auto-assign an available code as approved
    if pendingCount == 0 {
        var available models.SponCode
        if err := db.Where("team_id IS NULL AND code IS NOT NULL").First(&available).Error; err == nil {
            available.TeamID = &teamID
            available.Status = models.StatusApproved
            available.RequestedAt = time.Now().UTC()
            if err := db.Save(&available).Error; err != nil {
                return fiber.NewError(fiber.StatusInternalServerError, err.Error())
            }
            return c.Status(fiber.StatusOK).JSON(fiber.Map{
                "code":         available.Code,
                "team_id":      available.TeamID,
                "requested_at": available.RequestedAt,
                "status":       available.Status,
            })
        } else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
            return fiber.NewError(fiber.StatusInternalServerError, err.Error())
        }
        // If no available codes, fall through to create a pending entry
    }

    // Enforce: no more than 3 pending sponsor code rows per team
    if pendingCount >= 3 {
        return fiber.NewError(fiber.StatusBadRequest, "max pending sponsor code requests (3) reached for team")
    }

    // Create a pending entry with no code assigned
    var codePtr *string = nil
    row := models.SponCode{
        ID:          uuid.New(),
        Code:        codePtr,
        TeamID:      &teamID,
        Status:      models.StatusPending,
        RequestedAt: time.Now().UTC(),
    }
    if err := db.Create(&row).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }

    // Do not return code here; only acknowledge request
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "team_id":      row.TeamID,
        "requested_at": row.RequestedAt,
        "status":       row.Status,
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

    if err := db.Where("team_id = ? AND status = ?", *user.TeamID, models.StatusApproved).Find(&codes).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"codes": codes,
	})
}

// List pending sponsor code requests where code is null (admin only)
func ListPendingRequests(c *fiber.Ctx) error {
    user, ok := c.Locals("user").(models.User)
    if !ok {
        return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
    }
    if user.Role != models.RoleAdmin {
        return fiber.NewError(fiber.StatusForbidden, "forbidden")
    }

    db := initializer.Database.Db
    var rows []models.SponCode
    if err := db.Where("status = ? AND code IS NULL AND team_id IS NOT NULL", models.StatusPending).
        Preload("Team", func(tx *gorm.DB) *gorm.DB {
            return tx.Select([]string{"id", "name"})
        }).
        Order("requested_at ASC").
        Find(&rows).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }
    return c.Status(fiber.StatusOK).JSON(fiber.Map{"requests": rows})
}

// List available unowned codes (admin only)
func ListAvailableCodes(c *fiber.Ctx) error {
    user, ok := c.Locals("user").(models.User)
    if !ok {
        return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
    }
    if user.Role != models.RoleAdmin {
        return fiber.NewError(fiber.StatusForbidden, "forbidden")
    }

    db := initializer.Database.Db
    var rows []models.SponCode
    if err := db.Where("team_id IS NULL AND code IS NOT NULL").
        // Order("created_at ASC").
        Find(&rows).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }
    return c.Status(fiber.StatusOK).JSON(fiber.Map{"codes": rows})
}

// AdminAssignAvailableCode attaches an available code to a team and approves it (admin only)
func AdminAssignAvailableCode(c *fiber.Ctx) error {
    user, ok := c.Locals("user").(models.User)
    if !ok {
        return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
    }
    if user.Role != models.RoleAdmin {
        return fiber.NewError(fiber.StatusForbidden, "forbidden")
    }

    var body struct {
        Code   string     `json:"code"`
        TeamID *uuid.UUID `json:"team_id"`
    }
    if err := c.BodyParser(&body); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
    }
    if strings.TrimSpace(body.Code) == "" || body.TeamID == nil || *body.TeamID == uuid.Nil {
        return fiber.NewError(fiber.StatusBadRequest, "code and team_id are required")
    }

    db := initializer.Database.Db

    // Check approved cap for team
    var approvedCount int64
    if err := db.Model(&models.SponCode{}).
        Where("team_id = ? AND status = ?", *body.TeamID, models.StatusApproved).
        Count(&approvedCount).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }
    if approvedCount >= MAX_APPROVED_CODES_PER_TEAM {
        return fiber.NewError(fiber.StatusBadRequest,
            fmt.Sprintf("max approved codes (%d) reached for team", MAX_APPROVED_CODES_PER_TEAM))
    }

    var sc models.SponCode
    code := strings.ToUpper(strings.TrimSpace(body.Code))
    if err := db.Where("code = ?", code).First(&sc).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return fiber.NewError(fiber.StatusNotFound, "code not found")
        }
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }
    if sc.TeamID != nil {
        return fiber.NewError(fiber.StatusBadRequest, "code already assigned to a team")
    }

    sc.TeamID = body.TeamID
    sc.Status = models.StatusApproved
    sc.RequestedAt = time.Now().UTC()
    if err := db.Save(&sc).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "code":    sc.Code,
        "team_id": sc.TeamID,
        "status":  sc.Status,
    })
}
