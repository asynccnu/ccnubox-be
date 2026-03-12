package DO

import "time"

type FutureRecord struct {
	StuID    string `gorm:"column:stu_id;size:20;not null;primaryKey"`
	ID       string `gorm:"column:remote_id;size:64;not null;index:idx_future_remote_id"`
	Owner    string `gorm:"column:owner;size:50;not null"`
	Start    string `gorm:"column:start;size:32;not null;primaryKey"`
	End      string `gorm:"column:end;size:32;not null;primaryKey"`
	TimeDesc string `gorm:"column:time_desc;size:64"`
	States   string `gorm:"column:states;size:64"`
	DevName  string `gorm:"column:dev_name;size:100"`
	RoomID   string `gorm:"column:room_id;size:100"`
	RoomName string `gorm:"column:room_name;size:150"`
	LabName  string `gorm:"column:lab_name;size:100"`
}

func (FutureRecord) TableName() string {
	return "lib_future_records"
}

type HistoryRecord struct {
	StuID      string `gorm:"column:stu_id;size:20;not null;primaryKey"`
	Place      string `gorm:"column:place;size:100;not null"`
	Floor      string `gorm:"column:floor;size:50"`
	Status     string `gorm:"column:status;size:20"`
	Date       string `gorm:"column:date;size:20;not null"`
	SubmitTime string `gorm:"column:submit_time;size:32;not null;primaryKey"`
}

func (HistoryRecord) TableName() string {
	return "lib_history_records"
}

type Record struct {
	ID        string    `gorm:"primary key;column:remote_id;size:64;not null;index:idx_remote_id"`
	StuID     string    `gorm:"column:stu_id;size:20;not null"`
	SeatID    string    `gorm:"column:seat_id;size:64;index"`
	RoomID    string    `gorm:"column:room_id;size:64;index"`
	RoomName  string    `gorm:"column:room_name;size:100"`
	BuildName string    `gorm:"column:build_name;size:100"`
	FloorName string    `gorm:"column:floor_name;size:100"`
	SeatLabel string    `gorm:"column:seat_label;size:20"`
	MakeBegin time.Time `gorm:"column:make_begin;not null;index"`
	MakeEnd   time.Time `gorm:"column:make_end;not null"`
	MakeDate  time.Time `gorm:"column:make_date;type:date;not null;size:20"`
	Message   string    `gorm:"column:message;type:text"`
	Status    string    `gorm:"column:status;size:20;index"`
}

func (Record) TableName() string {
	return "lib_seat_reservation_records"
}
