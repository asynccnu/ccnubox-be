package grpc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

func (c *ContentServiceServer) GetSemester(ctx context.Context, in *contentv1.GetSemesterRequest) (*contentv1.GetSemesterResponse, error) {
	//如果没传date默认是当前日期
	date := in.GetDate()
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	semester, err := c.svcSemester.Get(ctx, date)
	if err != nil {
		return nil, err
	}
	return &contentv1.GetSemesterResponse{
		Semester: &contentv1.Semester{
			Semester:  semester.Semester,
			StartDate: semester.StartDate,
			EndDate:   semester.EndDate,
		},
	}, nil
}

func (c *ContentServiceServer) SaveSemester(ctx context.Context, in *contentv1.SaveSemesterRequest) (*contentv1.SaveSemesterResponse, error) {
	semester := domain.Semester{
		Semester:  in.GetSemester().GetSemester(),
		StartDate: in.Semester.GetStartDate(),
		EndDate:   in.Semester.GetEndDate(),
	}
	err := c.svcSemester.Save(ctx, &semester)
	if err != nil {
		return nil, err
	}
	return &contentv1.SaveSemesterResponse{}, nil

}

func (c *ContentServiceServer) GetSemesterList(ctx context.Context, in *contentv1.GetSemesterListRequest) (*contentv1.GetSemesterListResponse, error) {
	semesters, err := c.svcSemester.GetAll(ctx, in.GetStudentId())
	if err != nil {
		return nil, err
	}

	resp := &contentv1.GetSemesterListResponse{
		Semesters: make([]*contentv1.Semester, 0, len(semesters)),
	}

	for _, semester := range semesters {
		resp.Semesters = append(resp.Semesters, &contentv1.Semester{
			Semester:  semester.Semester,
			StartDate: semester.StartDate,
			EndDate:   semester.EndDate,
		})
	}

	return resp, nil
}
