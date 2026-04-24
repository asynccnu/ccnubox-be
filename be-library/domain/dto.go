package domain

type GetSeatInfosReq struct {
	StudentID string   `json:"studentId"`
	Rooms     []string `json:"rooms"`
}
