package domain

type GetGradeByTermReq struct {
	StudentID string   `json:"studentId"`
	Terms     []Term   `json:"terms"`
	Kcxzmcs   []string `json:"kcxzmc"`
	Refresh   bool     `json:"refresh" `
}

type Term struct {
	Xnm  int64   `json:"xnm"`  // 学年名，如 2024 表示 2024-2025 学年
	Xqms []int64 `json:"xqms"` // 学期列表
}
