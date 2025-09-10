package githubcontroller

import (
    "c2cbackend/initializer"
    "c2cbackend/models"
    "errors"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type SaveInstallationReq struct {
    InstallationID string  `json:"installation_id"`
    RepoFullName   *string `json:"repo_full_name"`
}

func SaveInstallation(c *fiber.Ctx) error {
    var body SaveInstallationReq
    if err := c.BodyParser(&body); err != nil {
        return &fiber.Error{Code: fiber.StatusBadRequest, Message: "invalid request body"}
    }
    if body.InstallationID == "" {
        return &fiber.Error{Code: fiber.StatusBadRequest, Message: "installation_id is required"}
    }

    user, ok := c.Locals("user").(models.User)
    if !ok {
        return &fiber.Error{Code: fiber.StatusUnauthorized, Message: "unauthorized"}
    }
    if user.TeamID == nil || *user.TeamID == uuid.Nil {
        return &fiber.Error{Code: fiber.StatusBadRequest, Message: "user has no team"}
    }

    db := initializer.Database.Db
    var team models.Team
    if err := db.First(&team, "id = ?", user.TeamID).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return &fiber.Error{Code: fiber.StatusNotFound, Message: "team not found"}
        }
        return &fiber.Error{Code: fiber.StatusInternalServerError, Message: "failed to load team"}
    }

    team.GithubInstallationID = body.InstallationID
    if body.RepoFullName != nil {
        team.GithubRepoFullName = *body.RepoFullName
    }

    if err := db.Save(&team).Error; err != nil {
        return &fiber.Error{Code: fiber.StatusInternalServerError, Message: "failed to save team"}
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message":            "saved",
        "installation_id":    team.GithubInstallationID,
        "repo_full_name":     team.GithubRepoFullName,
        "team_id":            team.ID,
    })
}

