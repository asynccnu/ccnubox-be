package model

type WrapClassInfo []*ClassInfo

func (w WrapClassInfo) ConvertToClass() ([]*Class, []string) {
	if len(w) == 0 {
		return nil, nil
	}
	Jxbmp := make(map[string]struct{})
	classes := make([]*Class, 0, len(w))
	for _, classInfo := range w {
		//thisWeek := classInfo.SearchWeek(week)
		class := &Class{
			Info: classInfo,
			//ThisWeek: thisWeek && tool.CheckIfThisYear(classInfo.Year, classInfo.Semester),
		}
		if classInfo.JxbId != "" {
			Jxbmp[classInfo.JxbId] = struct{}{}
		}
		classes = append(classes, class)
	}
	jxbIDs := make([]string, 0, len(Jxbmp))
	for k := range Jxbmp {
		jxbIDs = append(jxbIDs, k)
	}
	return classes, jxbIDs
}
