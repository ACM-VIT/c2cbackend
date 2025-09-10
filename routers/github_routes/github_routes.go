package github_routes

import (
    githubcontroller "c2cbackend/controllers/github_controller"
    "github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
    g := r.Group("/github")
    g.Post("/installation", githubcontroller.SaveInstallation)
}

