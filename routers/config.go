package routers

import (
	"c2cbackend/routers/review_routers"
	"c2cbackend/routers/round_routers"
	"c2cbackend/routers/team_routers"
	"c2cbackend/routers/track_routers"
	"c2cbackend/routers/user_routers"

	"github.com/gofiber/fiber/v2"
)

func PubSet(r fiber.Router) {
	user_routers.SetUp(r)
}

func Setup(r fiber.Router) {
	track_routers.Setup(r)
	team_routers.SetUp(r)
	round_routers.SetUp(r)
	review_routers.SetUp(r)
}
