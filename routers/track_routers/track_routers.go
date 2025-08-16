package track_routers

import (
	trackcontroller "c2cbackend/controllers/track_controller"

	"github.com/gofiber/fiber/v2"
)

func Setup(r fiber.Router) {
	r.Post("/tracks", trackcontroller.CreateTrack)
	r.Get("/tracks", trackcontroller.GetTracks)
	r.Put("/tracks/:trackid", trackcontroller.UpdateTrack)
	r.Delete("/tracks/:trackid", trackcontroller.DeleteTrack)
}
