package team_routers

import (
	teamcontroller "c2cbackend/controllers/team_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	team := r.Group("/team")
	team.Post("/create", teamcontroller.CreateTeam)
	team.Post("/join", teamcontroller.JoinTeamByCode)
	team.Get("/leave", teamcontroller.LeaveTeam)
	team.Post("/submission", teamcontroller.CreateTeamSubmission)
}
