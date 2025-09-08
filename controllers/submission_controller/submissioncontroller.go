package submissioncontroller

import (
    "c2cbackend/initializer"
    "c2cbackend/models"
    "net/http"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
)

// Admin and Reviewer can view all submissions
func GetAll(c *fiber.Ctx) error {
    u, ok := c.Locals("user").(models.User)
    if !ok || (u.Role != models.RoleAdmin && u.Role != models.RoleReviewer) {
        return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
    }
    var subs []models.Submission
    if err := initializer.Database.Db.Preload("Team").Preload("Round").Find(&subs).Error; err != nil {
        return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch submissions"})
    }
    return c.Status(http.StatusOK).JSON(fiber.Map{"submissions": subs})
}

// Admin and Reviewer can view submission by id
func GetByID(c *fiber.Ctx) error {
    u, ok := c.Locals("user").(models.User)
    if !ok || (u.Role != models.RoleAdmin && u.Role != models.RoleReviewer) {
        return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
    }
    sid, err := uuid.Parse(c.Params("id"))
    if err != nil {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
    }
    var sub models.Submission
    if err := initializer.Database.Db.Preload("Team").Preload("Round").First(&sub, "id = ?", sid).Error; err != nil {
        return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "not found"})
    }
    return c.Status(http.StatusOK).JSON(fiber.Map{"submission": sub})
}
