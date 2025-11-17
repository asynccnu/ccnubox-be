package DO

type CreditSummary struct {
	StuID  string `gorm:"column:stu_id;size:20;not null;primaryKey"`
	System string `gorm:"column:system;size:100;not null"`
	Remain string `gorm:"column:remain;size:100;not null"`
	Total  string `gorm:"column:total;size:100;not null"`
}

func (CreditSummary) TableName() string {
	return "lib_credit_summary"
}

type CreditRecord struct {
	StuID    string `gorm:"column:stu_id;size:20;not null;primaryKey"`
	Title    string `gorm:"column:title;size:100;not null"`
	Subtitle string `gorm:"column:subtitle;size:150;not null"`
	Location string `gorm:"column:location;size:100;not null"`
}

func (CreditRecord) TableName() string {
	return "lib_credit_record"
}
