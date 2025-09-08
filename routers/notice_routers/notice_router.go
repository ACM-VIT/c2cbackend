package noticerouters

import (
	noticecontroller "c2cbackend/controllers/notice_controller"
	
	"github.com/gofiber/fiber/v2"
)

func SetUp(r fiber.Router) {
	notices := r.Group("/notices")
	notices.Get("/", noticecontroller.GetNotices)
	notices.Post("/", noticecontroller.PostNotice)
	notices.Delete("/:id", noticecontroller.DeleteNotice)
}