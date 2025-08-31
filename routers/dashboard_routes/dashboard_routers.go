package dashboard_routes

import (
	dashboardcontroller "c2cbackend/controllers/dashboard_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	dashboard := r.Group("/dashboard")
	dashboard.Get("/", dashboardcontroller.Dashboard)
}
