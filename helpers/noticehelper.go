package helpers

type CreateNoticeReq struct {
	Information string `json:"information" valid:"required~Information is required"`
}