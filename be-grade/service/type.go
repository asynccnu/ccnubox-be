package service

import (
	"math"
	"strconv"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-grade/crawler"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
)

const (
	RegularGradePercentMSG = "平时成绩缺失"
	FinalGradePercentMAG   = "期末成绩缺失"
)

func modelConvDomain(grades []model.Grade) []domain.Grade {
	// 预分配切片容量，避免动态扩容
	domainGrades := make([]domain.Grade, 0, len(grades))

	// 遍历 grades，转换为 domain.Grade
	for _, grade := range grades {
		domainGrade := domain.Grade{
			StudentId:           grade.StudentId,
			Xnm:                 grade.Xnm,
			Xqm:                 grade.Xqm,
			KcId:                grade.KcId,  //课程id
			JxbId:               grade.JxbId, //教学班id
			Kcmc:                grade.Kcmc,  // 课程名称
			Xf:                  grade.Xf,    // 学分
			Cj:                  grade.Cj,
			Kcxzmc:              grade.Kcxzmc, // 课程性质名称
			Kclbmc:              grade.Kclbmc,
			Kcbj:                grade.Kcbj,                // 课程班级
			Jd:                  grade.Jd,                  // 绩点
			RegularGradePercent: grade.RegularGradePercent, // 平时成绩占比
			RegularGrade:        grade.RegularGrade,        // 平时成绩
			FinalGradePercent:   grade.FinalGradePercent,   // 期末成绩占比
			FinalGrade:          grade.FinalGrade,          // 期末成绩
		}

		// 将转换后的 domainGrade 加入切片
		domainGrades = append(domainGrades, domainGrade)
	}

	return domainGrades
}

func modelConvDomainAndFilter(grades []model.Grade, terms []domain.Term, kcxzmcs []string) []domain.Grade {
	// 如果 terms 为空，则跳过学年和学期筛选
	if len(terms) != 0 {

		// 学年筛选
		grades = filterByYear(grades, terms)

		// 学期筛选
		grades = filterByTerm(grades, terms)
	}

	grades = FilterByKcxzmc(grades, kcxzmcs)

	// 根据课程性质筛选
	return modelConvDomain(grades)
}

// 按学年筛选
func filterByYear(grades []model.Grade, terms []domain.Term) []model.Grade {
	termMap := make(map[int64]map[int64]struct{})
	for _, term := range terms {
		if _, exists := termMap[term.Xnm]; !exists {
			termMap[term.Xnm] = make(map[int64]struct{})
		}
	}

	// 筛选
	filtered := make([]model.Grade, 0)
	for _, grade := range grades {
		if _, ok := termMap[grade.Xnm]; ok {
			filtered = append(filtered, grade)
		}
	}
	return filtered
}

// 按学期筛选
func filterByTerm(grades []model.Grade, terms []domain.Term) []model.Grade {
	termMap := make(map[int64]map[int64]struct{})
	for _, term := range terms {
		if _, exists := termMap[term.Xnm]; !exists {
			termMap[term.Xnm] = make(map[int64]struct{})
		}
		for _, xqm := range term.Xqms {
			termMap[term.Xnm][xqm] = struct{}{}
		}
	}

	// 筛选
	filtered := make([]model.Grade, 0)
	for _, grade := range grades {
		if xqms, ok := termMap[grade.Xnm]; ok {
			if _, ok2 := xqms[grade.Xqm]; ok2 {
				filtered = append(filtered, grade)
			}
		}
	}

	return filtered
}

// 根据课程性质筛选
func FilterByKcxzmc(grades []model.Grade, kcxzmcs []string) []model.Grade {
	if len(kcxzmcs) == 0 {
		return grades
	}

	kcxzmcSet := make(map[string]struct{})
	for _, k := range kcxzmcs {
		kcxzmcSet[k] = struct{}{}
	}

	filtered := make([]model.Grade, 0)
	for _, grade := range grades {
		// 如果课程性质列表为空，则不过滤课程性质
		if _, ok := kcxzmcSet[grade.Kcxzmc]; ok {
			filtered = append(filtered, grade)
		}
	}

	return filtered
}

func aggregateGradeScore(grades []model.Grade) []domain.TypeOfGradeScore {
	// 定义一个 map，键是课程分类（例如课程ID），值是该分类下的所有成绩信息
	gradeMap := make(map[string][]domain.GradeScore)

	// 遍历所有成绩记录
	for _, grade := range grades {

		if grade.Cj < 60 {
			continue
		}

		// 使用课程性质作为 key
		key := grade.Kcxzmc

		// 如果当前课程分类不存在于 map 中，则初始化
		if _, exists := gradeMap[key]; !exists {
			gradeMap[key] = []domain.GradeScore{}
		}

		// 添加到当前课程的 Scores 列表
		gradeMap[key] = append(gradeMap[key], domain.GradeScore{
			Kcmc: grade.Kcmc,
			Xf:   grade.Xf,
		})
	}

	// 将 map 转换为切片形式返回
	result := make([]domain.TypeOfGradeScore, 0, len(gradeMap))
	for key, value := range gradeMap {
		result = append(result, domain.TypeOfGradeScore{
			Kcxzmc:         key,
			GradeScoreList: value,
		})
	}

	return result
}

func aggregateGrade(grades []crawler.Grade, details map[string]crawler.Score) []model.Grade {
	var result = make([]model.Grade, len(grades))
	for i, grade := range grades {
		// 解析学年和学期
		var xnm int
		var xqm int
		parts := strings.Split(grade.XQMC, "-")
		if len(parts) == 3 {
			xnm, _ = strconv.Atoi(parts[0])
			xqm, _ = strconv.Atoi(parts[2])
		}

		// 计算绩点
		jd := calcJd(grade.ZCJ)
		key := grade.XS0101ID + grade.JX0404ID
		detail := details[key]
		if detail.Cjxm1bl == "" && detail.Cjxm3bl == "" && detail.Cjxm1 == 0 && detail.Cjxm3 == 0 {
			detail.Cjxm1bl = FinalGradePercentMAG
			detail.Cjxm3bl = RegularGradePercentMSG
			detail.Cjxm1 = 0
			detail.Cjxm3 = 0
			details[key] = detail
		}

		result[i] = model.Grade{
			StudentId:           grade.XS0101ID,
			JxbId:               grade.JX0404ID,
			Kcmc:                grade.KCMC,
			Xnm:                 int64(xnm),
			Xqm:                 int64(xqm),
			Xf:                  grade.XF,
			Kcxzmc:              grade.KCXZMC,
			Kclbmc:              grade.KCSX,
			Kcbj:                "是否辅修字段暂时缺失",
			Jd:                  jd,
			RegularGradePercent: detail.Cjxm3bl,
			RegularGrade:        detail.Cjxm3,
			FinalGradePercent:   detail.Cjxm1bl,
			FinalGrade:          detail.Cjxm1,
			Cj:                  grade.ZCJ,
		}
	}
	return result
}

// 根据成绩计算绩点
func calcJd(score float32) float32 {
	if score >= 95 {
		return 4.5
	}
	if score < 60 {
		return 0
	}
	// 每下降5分绩点下降0.5
	diff := int(math.Ceil(float64((95 - score) / 5)))
	return 4.5 - float32(diff)*0.5
}

type FetchGrades struct {
	update []model.Grade
	final  []model.Grade
}

func ConvertGraduateGrade(graduateGrade []crawler.GraduatePoints) []model.Grade {
	var grades []model.Grade
	for _, p := range graduateGrade {
		var xqm int64
		switch p.Xqm {
		case "3":
			xqm = 1
		case "12":
			xqm = 2
		case "16":
			xqm = 3
		}
		grades = append(grades, model.Grade{
			StudentId: p.Xh,
			JxbId:     p.JxbID,
			Kcmc:      p.Kcmc,
			Xnm:       parseInt64(p.Xnm),
			Xqm:       xqm,
			Xf:        parseFloat32(p.Xf),
			Kcxzmc:    p.Kcxzmc,
			Kclbmc:    p.Kclbmc,
			Kcbj:      p.Kcbj,
			Jd:        parseFloat32(p.Jd),
			Cj:        parseFloat32(p.Cj),
		})
	}
	return grades
}

// parseInt64 辅助函数，将字符串转换为 int64
func parseInt64(value string) int64 {
	if i, err := strconv.Atoi(value); err == nil {
		return int64(i)
	}
	return 0
}

// parseFloat32 辅助函数，将字符串转换为 float32
func parseFloat32(value string) float32 {
	if i, err := strconv.ParseFloat(value, 32); err == nil {
		return float32(i)
	}
	return 0
}

func modelGraduateConvDomain(grades []model.Grade) []domain.Grade {
	res := make([]domain.Grade, 0, len(grades))
	for _, g := range grades {
		res = append(res, domain.Grade{
			Xnm:    g.Xnm,
			Xqm:    g.Xqm,
			JxbId:  g.JxbId,
			Kcmc:   g.Kcmc,
			Xf:     g.Xf,
			Cj:     g.Cj,
			Kcxzmc: g.Kcxzmc,
			Kclbmc: g.Kclbmc,
			Kcbj:   g.Kcbj,
			Jd:     g.Jd,
		})
	}
	return res
}
