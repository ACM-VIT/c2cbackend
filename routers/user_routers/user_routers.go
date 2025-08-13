package user_routers

import (
	usercontroller "c2cbackend/controllers/user_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	auth := r.Group("/auth")
	auth.Post("/signup", usercontroller.SignUp)
	auth.Get("/signin", usercontroller.SignIn)
}
