package tool

import (
	"testing"
	"time"
)

func TestGetXnmAndXqm(t *testing.T) {
	type args struct {
		currentTime time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantXnm string
		wantXqm string
	}{
		{
			name:    "Test in October",
			args:    args{currentTime: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)},
			wantXnm: "2023",
			wantXqm: "1",
		},
		{
			name:    "Test in January",
			args:    args{currentTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			wantXnm: "2022",
			wantXqm: "1",
		},
		{
			name:    "Test in March",
			args:    args{currentTime: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)},
			wantXnm: "2022",
			wantXqm: "2",
		},
		{
			name:    "Test in July",
			args:    args{currentTime: time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)},
			wantXnm: "2022",
			wantXqm: "3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotXnm, gotXqm := GetXnmAndXqm(tt.args.currentTime)
			if gotXnm != tt.wantXnm {
				t.Errorf("GetXnmAndXqm() gotXnm = %v, want %v", gotXnm, tt.wantXnm)
			}
			if gotXqm != tt.wantXqm {
				t.Errorf("GetXnmAndXqm() gotXqm = %v, want %v", gotXqm, tt.wantXqm)
			}
		})
	}
}
