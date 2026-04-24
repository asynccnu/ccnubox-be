package tool

import (
	"time"
)

func ParseTimeToMinute(t time.Time) int {
	hhmm := t.Hour()*60 + t.Minute()
	return hhmm
}

func ParseTodayTimeStringToUnix(tstr string) (int64, error) {
	loc, _ := GetLocation()
	today := time.Now().In(loc).Format("2006-01-02")
	str := today + " " + tstr
	t, err := time.ParseInLocation("2006-01-02 15:04", str, loc)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func ParseDateStringToTime(date string) (time.Time, error) {
	loc, _ := GetLocation()
	dateTime, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, err
	}
	return dateTime, nil
}

func ParseTimeStringToTime(tstr string) (time.Time, error) {
	loc, _ := GetLocation()
	t, err := time.ParseInLocation("2006-01-02 15:04", tstr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func ParseTodayTimeStringToTime(tstr string) (time.Time, error) {
	loc, _ := GetLocation()
	today := time.Now().In(loc).Format("2006-01-02")
	str := today + " " + tstr
	t, err := time.ParseInLocation("2006-01-02 15:04", str, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func GetLocation() (*time.Location, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, err
	}
	return loc, nil
}

func IsSameDay(date1 time.Time, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
