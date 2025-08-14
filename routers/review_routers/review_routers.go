package review_routers

import (
	reviewcontroller "c2cbackend/controllers/review_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	reviews := r.Group("/reviews")
	reviews.Post("/post/:rno/:team_id", reviewcontroller.PostReview)
	reviews.Delete("/:rno/:team_id", reviewcontroller.DeleteReview)
	reviews.Get("/all", reviewcontroller.GetReviews)
}
