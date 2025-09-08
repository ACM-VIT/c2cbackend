package attendanceroutes

import (
	attendancecontroller "c2cbackend/controllers/attendance_controller"

	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	attendance := r.Group("/attendance")
	attendance.Post("/mark/:user_id", attendancecontroller.MarkAttendance)
	attendance.Get("/all", attendancecontroller.FilterAttendance)
}
