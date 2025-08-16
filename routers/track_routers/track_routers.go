package track_routers

import (
	trackcontroller "c2cbackend/controllers/track_controller"

	"github.com/gofiber/fiber/v2"
)

func Setup(r fiber.Router) {
	track := r.Group("/tracks")
	track.Post("/create", trackcontroller.CreateTrack)
	track.Get("/getall", trackcontroller.GetTracks)
	track.Put("/update/:trackid", trackcontroller.UpdateTrack)
	track.Delete("/delete/:trackid", trackcontroller.DeleteTrack)
}
