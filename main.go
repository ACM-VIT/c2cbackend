package main

import (
	"c2cbackend/initializer"
	"c2cbackend/middleware"
	"c2cbackend/routers"
	"os"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {

	if os.Getenv("AUTH_USAGE") == "FIREBASE" {
		public := app.Group("/api/v1", middleware.FirebaseClaims())
		routers.PubSet(public)

		private := app.Group("/api/v1", middleware.FirebaseAuth())
		routers.Setup(private)
		private.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("Hello World!")
		})
	} else {
		public := app.Group("/api/v1", middleware.GoogleClaims())
		routers.PubSet(public)

		private := app.Group("/api/v1", middleware.GoogleAuth())
		routers.Setup(private)
		private.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("Hello World!")
		})
	}
}

func main() {
	initializer.InitFirebase()
	app := fiber.New()

	initializer.ConnectToDB()
	SetupRoutes(app)
	app.Listen(":3000")

}
