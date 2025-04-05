package data

import (
	"context"
	"github.com/asynccnu/ccnubox-be/classService/internal/model"
	"testing"
)

func initFCD() *FreeClassroomData {
	return &FreeClassroomData{
		cli: cli,
	}
}

func TestFreeClassroomData_AddClassroomOccupancy(t *testing.T) {
	fcd := initFCD()
	err := fcd.AddClassroomOccupancy(context.Background(), "2024", "1", model.CTWPair{
		CT: model.CTime{
			Day:      1,
			Sections: []int{1, 2},
			Weeks:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
		Where: "n109",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestFreeClassroomData_DeleteClassroomOccupancy(t *testing.T) {
	fcd := initFCD()
	//err := fcd.AddClassroomOccupancy(context.Background(), "2023", "1", model.CTWPair{
	//	CT: model.CTime{
	//		Day:      1,
	//		Sections: []int{1, 2},
	//		Weeks:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
	//	},
	//	Where: "n109",
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//_, err = fcd.cli.Refresh().Index(freeClassroomIndex).Do(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//} // 确保添加的数据可见
	err := fcd.ClearClassroomOccupancy(context.Background(), "2023", "1")
	if err != nil {
		t.Error(err)
	}
}

func TestFreeClassroomData_QueryAvailableClassrooms(t *testing.T) {
	fcd := initFCD()

	type args struct {
		year     string
		semester string
		pairs    []model.CTWPair
	}
	arg := args{
		year:     "2024",
		semester: "1",
		pairs: []model.CTWPair{
			{
				CT: model.CTime{
					Day:      1,
					Sections: []int{1, 2},
					Weeks:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
				Where: "n109",
			},
			{
				CT: model.CTime{
					Day:      2,
					Sections: []int{1, 2},
					Weeks:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
				Where: "n109",
			},
			{
				CT: model.CTime{
					Day:      1,
					Sections: []int{3, 4},
					Weeks:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
				Where: "n119",
			},
			{
				CT: model.CTime{
					Day:      1,
					Sections: []int{1, 2},
					Weeks:    []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
				Where: "n129",
			},
			{
				CT: model.CTime{
					Day:      1,
					Sections: []int{1, 2},
					Weeks:    []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				},
				Where: "7129",
			},
		},
	}

	for _, pair := range arg.pairs {
		err := fcd.AddClassroomOccupancy(context.Background(), arg.year, arg.semester, pair)
		if err != nil {
			t.Error(err)
		}
	}
	t.Run("test1", func(t *testing.T) {
		res, err := fcd.QueryAvailableClassrooms(context.Background(), arg.year, arg.semester, 12, 1, 1, "n1")
		if err != nil {
			t.Error(err)
		}
		t.Log(res)
	})

}
