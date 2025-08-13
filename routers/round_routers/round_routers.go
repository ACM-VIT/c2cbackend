package round_routers

import (
	roundcontroller "c2cbackend/controllers/round_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	round := r.Group("/round")
	round.Post("/", roundcontroller.CreateRound)
	round.Delete("/:rno", roundcontroller.DeleteRound)
	round.Put("/:rno", roundcontroller.UpdateRound)
	round.Get("/rankings/:rno", roundcontroller.GetRoundTeamRankings)
	round.Post("/:rno/promote", roundcontroller.PromoteToRound)
	round.Get("/:rno/assignall", roundcontroller.AssignAllToRound)
}
