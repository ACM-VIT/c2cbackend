package submission_routers

import (
    submissioncontroller "c2cbackend/controllers/submission_controller"
    "github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
    sub := r.Group("/submissions")
    sub.Get("/all", submissioncontroller.GetAll)
    sub.Get("/:id", submissioncontroller.GetByID)
}
