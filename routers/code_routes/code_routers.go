package code_routes

import (
	codecontroller "c2cbackend/controllers/code_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	code := r.Group("/code")
	code.Post("/seed", codecontroller.SeedCodesFromFile)
	code.Post("/request", codecontroller.RequestCode)
	code.Get("/team", codecontroller.GetTeamCodes)
	code.Post("/assign", codecontroller.AssignCode)
}
