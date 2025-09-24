package service

import (
	"strconv"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
)

//辅助函数

func aggregateGrades(detailItems []GetDetailItem, KcxzItems []GetKcxzItem) []model.Grade {
	// 2. 用于按 JxbId 进行聚合
	gradeMap := make(map[string]*model.Grade)
	//初始化gradeMap
	for _, item := range KcxzItems {
		// 以 JxbId 作为唯一 key 进行聚合
		key := item.JxbID
		// 如果当前 JxbId 不在 map 中，初始化 Grade 结构体
		if _, exists := gradeMap[key]; !exists {
			//进行转换
			var xqm int64
			switch item.Xqm {
			case "3":
				xqm = 1
			case "12":
				xqm = 2
			case "16":
				xqm = 3
			}
			gradeMap[key] = &model.Grade{
				Kcxzmc:              item.Kcxzmc,
				Kclbmc:              item.Kclbmc,
				Kcbj:                item.Kcbj,
				Studentid:           item.Xh,
				JxbId:               item.JxbID,
				Jd:                  parseFloat32(item.Jd),
				Kcmc:                item.Kcmc,
				Xnm:                 parseInt64(item.Xnm),
				Xqm:                 xqm,
				Xf:                  parseFloat32(item.Xf),
				Cj:                  parseFloat32(item.Cj),
				RegularGradePercent: "平时(0%)",
				FinalGradePercent:   "期末(0%)",
			}
		}
	}

	//二次遍历
	for _, item := range detailItems {
		// 以 JxbId 作为唯一 key 进行聚合
		key := item.JxbID
		// 如果当前 JxbId存在于gradeMap中那么进行判断并赋值
		if _, exists := gradeMap[key]; exists {
			// 根据 xmblmc 字段的内容分配字段值
			if strings.Contains(item.Xmblmc, "平时") {
				gradeMap[key].RegularGradePercent = item.Xmblmc
				gradeMap[key].RegularGrade = parseFloat32(item.Xmcj)
			} else if strings.Contains(item.Xmblmc, "期末") {
				gradeMap[key].FinalGradePercent = item.Xmblmc
				gradeMap[key].FinalGrade = parseFloat32(item.Xmcj)
			} else if strings.Contains(item.Xmblmc, "总评") {
				gradeMap[key].Cj = parseFloat32(item.Xmcj)
			}

		}

	}

	// 4. 将 map 转换为切片返回
	var grades []model.Grade
	for _, grade := range gradeMap {
		grades = append(grades, *grade)
	}

	return grades
}

func convertGraduateGrade(graduateGrade []GraduatePoints) []model.GraduateGrade {
	var grades []model.GraduateGrade
	for _, p := range graduateGrade {
		// 学期转换
		var term int64
		switch p.Xqm {
		case "3":
			term = 1
		case "12":
			term = 2
		case "16":
			term = 3
		}

		grades = append(grades, model.GraduateGrade{
			JxbId:           p.JxbID,
			Status:          p.Cjztmc,
			Year:            p.Xnmmc,
			Term:            term,
			StudentID:       p.Xh,
			Name:            p.Xm,
			StudentCategory: p.Xslbmc,
			College:         p.Jgmc,
			Major:           p.Zymc,
			Grade:           parseInt64(p.NjdmID),
			ClassCode:       p.KchID,
			ClassName:       p.Kcmc,
			ClassNature:     p.Kcxzmc,
			Credit:          parseFloat32(p.Xf),
			Point:           parseFloat32(p.Cj),
			GradePoints:     parseFloat32(p.Jd),
			IsAvailable:     p.Cjsfzf,
			IsDegree:        p.Sfxwkc,
			SetCollege:      p.Kkbmmc,
			ClassMark:       p.Kcbj,
			ClassCategory:   p.Kclbmc,
			ClassID:         p.JxbID,
			Teacher:         p.Jsxm,
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

func modelConvDomain(grades []model.Grade) []domain.Grade {
	// 预分配切片容量，避免动态扩容
	domainGrades := make([]domain.Grade, 0, len(grades))

	// 遍历 grades，转换为 domain.Grade
	for _, grade := range grades {
		domainGrade := domain.Grade{
			Xnm:                 grade.Xnm,
			Xqm:                 grade.Xqm,
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

func modelGraduateConvDomain(grades []model.GraduateGrade) []domain.GraduateGrade {
	res := make([]domain.GraduateGrade, 0, len(grades))
	for _, g := range grades {
		res = append(res, domain.GraduateGrade{
			StudentID:       g.StudentID,
			JxbId:           g.JxbId,
			Status:          g.Status,
			Year:            g.Year,
			Term:            g.Term,
			Name:            g.Name,
			StudentCategory: g.StudentCategory,
			College:         g.College,
			Major:           g.Major,
			Grade:           g.Grade,
			ClassCode:       g.ClassCode,
			ClassName:       g.ClassName,
			ClassNature:     g.ClassNature,
			Credit:          g.Credit,
			Point:           g.Point,
			GradePoints:     g.GradePoints,
			IsAvailable:     g.IsAvailable,
			IsDegree:        g.IsDegree,
			SetCollege:      g.SetCollege,
			ClassMark:       g.ClassMark,
			ClassCategory:   g.ClassCategory,
			ClassID:         g.ClassID,
			Teacher:         g.Teacher,
		})
	}
	return res
}
