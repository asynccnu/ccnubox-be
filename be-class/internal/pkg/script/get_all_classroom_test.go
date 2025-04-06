package script

import "testing"

func TestGetAllClassRooms(t *testing.T) {
	cookie := "JSESSIONID=8425A082C7AC107B9EDD4FA1AF4FAA1F"
	err := GetAllClassRooms("2024", "2", cookie)
	if err != nil {
		t.Fatal(err)
	}
}
