package tool

import (
	"reflect"
	"testing"
	"time"
)

func TestCheckSY(t *testing.T) {
	type args struct {
		semester string
		year     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test1", args: args{"1", "2023-2024"}, want: false},
		{name: "test2", args: args{"2", "2023"}, want: true},
		{name: "test3", args: args{"3", "2023-2024"}, want: false},
		{name: "test4", args: args{"1", "-2024"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckSY(tt.args.semester, tt.args.year); got != tt.want {
				t.Errorf("CheckSY() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatWeeks(t *testing.T) {
	type args struct {
		weeks []int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "连续的周数",
			args: args{weeks: []int{1, 2, 3, 4}},
			want: "1-4周",
		},
		{
			name: "间隔的周数",
			args: args{weeks: []int{1, 3, 4}},
			want: "1,3-4周",
		},
		{
			name: "单周",
			args: args{weeks: []int{1, 3, 5}},
			want: "1,3,5周(单)",
		},
		{
			name: "双周",
			args: args{weeks: []int{2, 4, 6}},
			want: "2,4,6周(双)",
		},
		{
			name: "混合周数",
			args: args{weeks: []int{1, 2, 4, 6}},
			want: "1-2,4,6周",
		},
		{
			name: "单个周数",
			args: args{weeks: []int{1}},
			want: "1周(单)",
		},
		{
			name: "两个连续周数",
			args: args{weeks: []int{2, 3}},
			want: "2-3周",
		},
		{
			name: "空周数集合",
			args: args{weeks: []int{}},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatWeeks(tt.args.weeks); got != tt.want {
				t.Errorf("FormatWeeks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWeeks(t *testing.T) {
	type args struct {
		weeks int64
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			name: "test1",
			args: args{weeks: 1}, // 0001
			want: []int{1},
		},
		{
			name: "test2",
			args: args{weeks: 3}, // 0011
			want: []int{1, 2},
		},
		{
			name: "test3",
			args: args{weeks: 5}, // 0101
			want: []int{1, 3},
		},
		{
			name: "test4",
			args: args{weeks: 15}, // 1111
			want: []int{1, 2, 3, 4},
		},
		{
			name: "test5",
			args: args{weeks: 16}, // 10000
			want: []int{5},
		},
		{
			name: "test6",
			args: args{weeks: 31}, // 11111
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "test7",
			args: args{weeks: 63}, // 111111
			want: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name: "test8",
			args: args{weeks: 0}, // 000000
			want: []int{},
		},
		{
			name: "test9",
			args: args{weeks: 42}, // 101010
			want: []int{2, 4, 6},
		},
		{
			name: "test10",
			args: args{weeks: 32767}, // 101011
			want: []int{8, 11, 13, 15},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseWeeks(tt.args.weeks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseWeeks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckIfThisWeek(t *testing.T) {
	type args struct {
		xnm string
		xqm string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{"2023", "3"}, false},
		{"test2", args{"2023", "1"}, true},
		{"test3", args{"2024", "1"}, false},
		{"test4", args{"2024", "2"}, false},
		{"test5", args{"2026", "1"}, false},
		{"test6", args{"2026", "2"}, false},
		{"test7", args{"2026", "3"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckIfThisYear(tt.args.xnm, tt.args.xqm); got != tt.want {
				t.Errorf("CheckIfThisYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomBool(t *testing.T) {
	for j := 1; j <= 10; j++ {
		cnt := 0
		N := 100
		for i := 1; i <= N; i++ {
			if RandomBool(0.3) {
				cnt++
			}
		}
		t.Logf("happened:%d/%d", cnt, N)
	}

}

func TestFormatTimeInUTC(t *testing.T) {
	// 定义测试用例
	tests := []struct {
		input    time.Time
		expected string
	}{
		{
			// 测试当前时间
			input:    time.Date(2025, 2, 21, 10, 30, 0, 123456000, time.Local),
			expected: "2025-02-21T02:30:00.123456",
		},
		{
			// 测试没有时间差的时间
			input:    time.Date(2025, 2, 21, 10, 30, 0, 0, time.Local),
			expected: "2025-02-21T02:30:00.000000",
		},
		{
			// 测试时间与 UTC 有时间差
			input:    time.Date(2025, 2, 21, 10, 30, 0, 123456000, time.UTC),
			expected: "2025-02-21T10:30:00.123456",
		},
	}

	// 遍历测试用例并执行每个测试
	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			// 调用要测试的函数
			got := FormatTimeInUTC(tt.input)

			// 验证返回的结果是否与预期相符
			if got != tt.expected {
				t.Errorf("FormatTimeInUTC(%v) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}
