package tool

import (
	"strconv"
	"time"
)

func GetXnmAndXqm(currentTime time.Time) (xnm, xqm string) {
	currentYear := currentTime.Year()
	currentMonth := currentTime.Month()
	//currentYear := 2023
	//currentMonth := 10
	if currentMonth >= 9 {
		xnm = strconv.Itoa(currentYear)
		xqm = "1"
	} else if currentMonth <= 1 {
		xnm = strconv.Itoa(currentYear - 1)
		xqm = "1"
	} else if currentMonth >= 2 && currentMonth <= 6 {
		xnm = strconv.Itoa(currentYear - 1)
		xqm = "2"
	} else {
		xnm = strconv.Itoa(currentYear - 1)
		xqm = "3"
	}
	return
}
