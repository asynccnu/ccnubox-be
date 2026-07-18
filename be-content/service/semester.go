package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/pkg/estimation"
	"github.com/asynccnu/ccnubox-be/be-content/repository"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	GET_SEMESTER_LIST_ERROR = errorx.FormatErrorFunc(contentv1.ErrorGetSemesterListError("获取所有学期失败"))
	GET_SEMESTER_ERROR      = errorx.FormatErrorFunc(contentv1.ErrorGetSemesterError("获取当前学期失败"))
	SAVE_SEMESTER_ERROR     = errorx.FormatErrorFunc(contentv1.ErrorSaveSemesterError("保存学期信息失败"))
)

type SemesterService interface {
	Get(ctx context.Context, t string) (*domain.Semester, error)
	GetAll(ctx context.Context, studentId string) ([]*domain.Semester, error)
	Save(ctx context.Context, s *domain.Semester) error
}

type semesterService struct {
	repo repository.ContentRepo[model.Semester]
	l    logger.Logger
}

func NewSemesterService(repo repository.ContentRepo[model.Semester], l logger.Logger) SemesterService {
	return &semesterService{repo: repo, l: l}
}

func (se *semesterService) GetAll(ctx context.Context, studentId string) ([]*domain.Semester, error) {
	admissionYear, err := extractAdmissionYear(studentId)
	if err != nil {
		return nil, GET_SEMESTER_LIST_ERROR(err)
	}

	dbSemesters, err := se.repo.GetList(ctx)
	if err != nil {
		return nil, GET_SEMESTER_LIST_ERROR(err)
	}
	dbMap := make(map[string]model.Semester, len(dbSemesters))
	for _, s := range dbSemesters {
		dbMap[s.Semester] = s
	}

	now := time.Now()
	academicYear, currentSemester := estimation.GetAcademicInfo(now)

	semesters := buildSemesterList(dbMap, admissionYear, academicYear, currentSemester)

	// 倒序（最新在前）
	for i, j := 0, len(semesters)-1; i < j; i, j = i+1, j-1 {
		semesters[i], semesters[j] = semesters[j], semesters[i]
	}
	return semesters, nil
}

func (se *semesterService) Get(ctx context.Context, date string) (*domain.Semester, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, GET_SEMESTER_ERROR(err)
	}

	dbSemesters, err := se.repo.GetList(ctx)
	if err != nil {
		return nil, GET_SEMESTER_ERROR(err)
	}

	if len(dbSemesters) == 0 {
		return calcFallbackSemester(t), nil
	}

	dbMap := make(map[string]model.Semester, len(dbSemesters))
	for _, s := range dbSemesters {
		dbMap[s.Semester] = s
	}

	// 找出 DB 中最小的学年，计算补充最小学年和当前学年中缺失的数据再匹配
	startYear := math.MaxInt
	for _, s := range dbSemesters {
		parts := strings.SplitN(s.Semester, "-", 2)
		if len(parts) != 2 {
			continue
		}
		if y, err := strconv.Atoi(parts[0]); err == nil && y < startYear {
			startYear = y
		}
	}
	if startYear == math.MaxInt {
		return calcFallbackSemester(t), nil
	}

	now := time.Now()
	academicYear, currentSemester := estimation.GetAcademicInfo(now)

	semesters := buildSemesterList(dbMap, startYear, academicYear, currentSemester)

	for _, s := range semesters {
		sd, err1 := time.Parse("2006-01-02", s.StartDate)
		ed, err2 := time.Parse("2006-01-02", s.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		if !t.Before(sd) && !t.After(ed) {
			return s, nil
		}
		if t.Before(sd) {
			return s, nil
		}
	}
	return calcFallbackSemester(t), nil
}

func (se *semesterService) Save(ctx context.Context, s *domain.Semester) error {
	startDate, err := time.Parse("2006-01-02", s.StartDate)
	if err != nil {
		return SAVE_SEMESTER_ERROR(err)
	}
	endDate, err := time.Parse("2006-01-02", s.EndDate)
	if err != nil {
		return SAVE_SEMESTER_ERROR(err)
	}

	//学期格式校验: 必须是 "YYYY-S" 格式，前四位2000~2999，后一位1~3
	matched, _ := regexp.MatchString(`^2\d{3}-[1-3]$`, s.Semester)
	if !matched {
		return SAVE_SEMESTER_ERROR(errorx.Errorf("invalid semester format: %s, must be like '2026-2', year 2000-2999, semester 1-3", s.Semester))
	}

	//添加时间合法性校验
	if startDate.After(endDate) {
		return SAVE_SEMESTER_ERROR(errorx.Errorf("save semester time invalid: start estimation:%s,end estimation:%s", s.StartDate, s.EndDate))
	}

	record, err := se.repo.Get(ctx, "semester", s.Semester)
	//记录不存在：创建新记录
	if errors.Is(err, repository.ErrRecordNotFound) {
		modelSemester := &model.Semester{
			Semester:  s.Semester,
			StartDate: startDate,
			EndDate:   endDate,
		}
		err = se.repo.Save(ctx, modelSemester)
		if err != nil {
			return SAVE_SEMESTER_ERROR(err)
		}
		return nil
	}
	if err != nil {
		return SAVE_SEMESTER_ERROR(err)
	}

	//如果存在记录，更新记录
	record.Semester = s.Semester
	record.StartDate = startDate
	record.EndDate = endDate
	record.UpdatedAt = time.Now()

	err = se.repo.Save(ctx, record)
	if err != nil {
		return SAVE_SEMESTER_ERROR(err)
	}
	return nil
}

// calcFallbackSemester 根据日期推算所属学期
func calcFallbackSemester(t time.Time) *domain.Semester {
	academicYear, semester := estimation.GetAcademicInfo(t)
	startDate, endDate := estimation.EstimateDateRange(academicYear, semester)
	return &domain.Semester{
		Semester:  fmt.Sprintf("%d-%d", academicYear, semester),
		StartDate: startDate,
		EndDate:   endDate,
	}
}

// buildSemesterList 生成从 startYear 到 academicYear 的完整学期列表，
// 优先使用 DB 数据填充日期，缺失的用 estimateDateRange 推算。
func buildSemesterList(dbMap map[string]model.Semester, startYear, academicYear, currentSemester int) []*domain.Semester {
	var semesters []*domain.Semester
	for year := startYear; year <= academicYear; year++ {
		for s := 1; s <= 3; s++ {
			if year == academicYear && s > currentSemester {
				break
			}
			key := fmt.Sprintf("%d-%d", year, s)
			sem := &domain.Semester{Semester: key}
			if dbS, ok := dbMap[key]; ok {
				sem.StartDate = dbS.StartDate.Format("2006-01-02")
				sem.EndDate = dbS.EndDate.Format("2006-01-02")
			} else {
				sem.StartDate, sem.EndDate = estimation.EstimateDateRange(year, s)
			}
			semesters = append(semesters, sem)
		}
	}
	return semesters
}

// extractAdmissionYear 从学号中提取入学年份
func extractAdmissionYear(studentId string) (int, error) {
	if len(studentId) < 4 {
		return 0, fmt.Errorf("invalid student ID: %s", studentId)
	}
	year, err := strconv.Atoi(studentId[:4])
	if err != nil || year <= 0 {
		return 0, fmt.Errorf("invalid student ID: %s", studentId)
	}
	return year, nil
}
