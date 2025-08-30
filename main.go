package main

import (
	"c2cbackend/initializer"
	"c2cbackend/middleware"
	"c2cbackend/routers"
	"os"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Hello World!",
		})
	})

	if os.Getenv("AUTH_USAGE") == "FIREBASE" {
		public := app.Group("/api/v1", middleware.FirebaseClaims())
		routers.PubSet(public)

		private := app.Group("/api/v1", middleware.FirebaseAuth())
		routers.Setup(private)

	} else {
		public := app.Group("/api/v1", middleware.GoogleClaims())
		routers.PubSet(public)

		private := app.Group("/api/v1", middleware.GoogleAuth())
		routers.Setup(private)

	}
}

func main() {
	initializer.InitFirebase()
	app := fiber.New()

	initializer.ConnectToDB()
	// CORS allow all
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "*")
		c.Set("Access-Control-Allow-Headers", "*")
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Next()
	})
	SetupRoutes(app)
	app.Listen(":8080")

}
