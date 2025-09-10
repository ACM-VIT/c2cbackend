package detailrouter

import (
	 "c2cbackend/controllers/detail_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	auth := r.Group("/detail")
	auth.Post("/room", detailcontroller.AddRoom)
}
