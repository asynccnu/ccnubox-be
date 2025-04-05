package domain

// FeedEvent的增加已读还是未读字段版本
type FeedEventVO struct {
	ID           int64             `json:"id"` // ID
	StudentId    string            `json:"student_id"`
	Type         string            `json:"type"`          // 类型
	Title        string            `json:"title"`         // 提示用的字段
	Content      string            `json:"content"`       // 正式文本
	ExtendFields map[string]string `json:"extend_fields"` // 拓展字段
	CreatedAt    int64             `json:"created_at"`    // 创建时间，Unix 时间戳（int格式）
	Read         bool              `json:"read"`
}
