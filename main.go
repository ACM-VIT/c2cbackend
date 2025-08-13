package main

import (
	"c2cbackend/initializer"
	"c2cbackend/middleware"
	"c2cbackend/routers"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	//JUGAAD FLOW NEEDS FIX (NO FUNCTIONALITY ISSUE , BUT WILL CHANGE IF TIME PERSISTS)
	public := app.Group("/api/v1", middleware.FirebaseClaims())
	routers.PubSet(public)

	private := app.Group("/api/v1", middleware.FirebaseAuth())
	routers.Setup(private)
	private.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello World!")
	})
}

func main() {
	initializer.InitFirebase()
	app := fiber.New()

	initializer.ConnectToDB()
	SetupRoutes(app)
	app.Listen(":3000")

}
