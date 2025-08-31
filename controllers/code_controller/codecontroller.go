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
	"gorm.io/gorm/clause"
)

func SeedCodesFromFile(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if user.Role != "admin" {
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

	var existing models.SponCode
	if err := db.Where("team_id = ?", teamID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":         existing.Code,
			"team_id":      existing.TeamID,
			"requested_at": existing.RequestedAt,
		})
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var assigned models.SponCode
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("team_id = ?", teamID).
			First(&existing).Error; err == nil {
			assigned = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var free models.SponCode
		q := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("team_id IS NULL").
			Order("requested_at NULLS FIRST, created_at ASC, code ASC").
			First(&free)

		if q.Error != nil {
			if errors.Is(q.Error, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusConflict, "no codes left to assign")
			}
			return q.Error
		}

		now := time.Now()
		free.TeamID = &teamID
		free.RequestedAt = now

		if err := tx.Save(&free).Error; err != nil {
			return err
		}

		assigned = free
		return nil
	}); err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":         assigned.Code,
		"team_id":      assigned.TeamID,
		"requested_at": assigned.RequestedAt,
	})
}
