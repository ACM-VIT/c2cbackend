package whitelistrouters

import (
	whitelistcontroller "c2cbackend/controllers/whitelist_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	r.Get("/whitelist", whitelistcontroller.CheckWhitelist)
}