package biz

import (
	"context"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	"github.com/asynccnu/ccnubox-be/be-class/internal/service"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

const (
	identity = "个性发展课程"
	common   = "通识教育课程"
	specific = "专业主干课程"

	new_ = "new"
)

var (
	courseURL         = "https://bkzhjw.ccnu.edu.cn/jsxsd/pyfa/topyfamx"
	classTypes        = []string{identity, common, specific}
	ErrRecordNotFound = gorm.ErrRecordNotFound
)

// yearTermMapper 第一类情况映射
var yearTermMapper = map[string]string{
	"1":  "第一学年第一学期",
	"2":  "第一学年第二学期",
	"3":  "第一学年第三学期",
	"4":  "第二学年第一学期",
	"5":  "第二学年第二学期",
	"6":  "第二学年第三学期",
	"7":  "第三学年第一学期",
	"8":  "第三学年第二学期",
	"9":  "第三学年第三学期",
	"10": "第四学年第一学期",
	"11": "第四学年第二学期",
	"12": "第四学年第三学期",
}

// yearTermMapper2 第二类情况映射
var yearTermMapper2 = map[string]string{
	"0-1": "1",
	"0-2": "2",
	"0-3": "3",
	"1-1": "4",
	"1-2": "5",
	"1-3": "6",
	"2-1": "7",
	"2-2": "8",
	"2-3": "9",
	"3-1": "10",
	"3-2": "11",
	"3-3": "12",
}

var classMapper = map[string]string{
	identity: "0",
	common:   "1",
	specific: "2",
}

type CultivateStrategyData interface {
	BatchSaveToBeStudiedClass(ctx context.Context, relations []model.UnStudiedClassStudentRelationship, classes []model.ToBeStudiedClass) error
	GetClassStudentRelation(ctx context.Context, stuId, status string, alive time.Duration) ([]model.UnStudiedClassStudentRelationship, error)
	GetDetailUnStudyClass(ctx context.Context, id string) (model.ToBeStudiedClass, error)

	DataAlive() time.Duration
}

type CultivateStrategyBiz struct {
	proxyCli  proxy.Client
	cookieCli CookieClient
	csData    CultivateStrategyData
	cache     Cache
}

func NewCultivateStrategyBiz(cookieCli CookieClient, cache Cache, csData CultivateStrategyData, proxyCli proxy.Client) service.CultivateStrategy {
	c := &CultivateStrategyBiz{
		cookieCli: cookieCli,
		cache:     cache,
		csData:    csData,
		proxyCli:  proxyCli,
	}

	return c
}

func (c *CultivateStrategyBiz) GetToBeStudiedClass(ctx context.Context, stuId, status string) (service.ToBeStudiedClasses, error) {
	var (
		res     service.ToBeStudiedClasses
		er, err error
	)
	// 优先使用数据库的新数据
	log.Debug("Trying to get fresh unstudied classes from DB")
	res, err = c.GetCultivateStrategyFromDB(ctx, stuId, status, c.csData.DataAlive())
	if err == nil {
		return res, nil
	}

	log.Debug("No fresh unstudied classes from DB, try crawl from ccnu")
	// 如果数据库里面没有, 去爬数据, 然后更新
	if errors.Is(err, ErrRecordNotFound) {
		res, er = c.GetCultivateStrategyFromCCNU(ctx, stuId)
		if er == nil {
			resCopy := res
			go func(classes service.ToBeStudiedClasses) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := c.SaveClassesAndRelationsToDB(ctx, stuId, &resCopy); err != nil {
					// TODO log
					fmt.Printf("gouroutine save err: %+v\n", err)
				}
			}(resCopy)

			// 如果是post的筛选, 这里对爬虫数据进行处理
			if status != "" {
				resp := findClassesByStatusFromRes(status, res)
				return resp, nil
			}

			return res, nil
		}
		log.Error("Get unstudied classes from CCNU err: ", er)
	}

	// 如果没有新数据 + 爬虫没爬到数据, 去查看旧数据
	res, err = c.GetCultivateStrategyFromDB(ctx, stuId, status, -1)
	if err == nil {
		return res, nil
	}

	log.Warn("No old unstudied classes from DB")
	return service.ToBeStudiedClasses{}, err
}

// aggregateToService 把数据库模型转换成领域模型, 但是感觉data层有更好的处理方法 TODO
func aggregateToService(r model.UnStudiedClassStudentRelationship, c model.ToBeStudiedClass) service.ToBeStudiedClass {
	return service.ToBeStudiedClass{
		Id:        c.Id,
		Name:      c.Name,
		Status:    r.Status,
		Property:  c.Property,
		Studiable: c.Studiable,
		Credit:    c.Credit,
		Type:      c.Type,
	}
}

// SaveClassesAndRelationsToDB 将爬虫结果存入数据库
func (c *CultivateStrategyBiz) SaveClassesAndRelationsToDB(ctx context.Context, stuId string, res *service.ToBeStudiedClasses) error {
	classes, relations := resToModels(stuId, res)
	return c.csData.BatchSaveToBeStudiedClass(ctx, relations, classes)
}

// resToModels 将爬虫结果转换为数据库模型
func resToModels(stuId string, res *service.ToBeStudiedClasses) ([]model.ToBeStudiedClass, []model.UnStudiedClassStudentRelationship) {
	classes := joinClasses(res)

	return separateClass(stuId, classes)
}

// joinClasses 因为爬虫结果分三个相同切片的字段, 这里要聚合一下
func joinClasses(res *service.ToBeStudiedClasses) []service.ToBeStudiedClass {
	var classes []service.ToBeStudiedClass
	classes = append(classes, res.IdentityDevelop...)
	classes = append(classes, res.CommonEducate...)
	classes = append(classes, res.SpecificSkill...)
	return classes
}

// separateClass 把爬虫结果每个字段的切片处理成数据库模型
func separateClass(stuId string, classes []service.ToBeStudiedClass) ([]model.ToBeStudiedClass, []model.UnStudiedClassStudentRelationship) {
	var (
		mcs []model.ToBeStudiedClass
		mrs []model.UnStudiedClassStudentRelationship
	)

	for _, c := range classes {
		mcs = append(mcs, classToModel(&c))
		mrs = append(mrs, classToRelation(stuId, &c))
	}

	return mcs, mrs
}

func classToModel(c *service.ToBeStudiedClass) model.ToBeStudiedClass {
	return model.ToBeStudiedClass{
		Id:        c.Id,
		Name:      c.Name,
		Property:  c.Property,
		Studiable: c.Studiable,
		Credit:    c.Credit,
		Type:      c.Type,
	}
}

func classToRelation(stuId string, c *service.ToBeStudiedClass) model.UnStudiedClassStudentRelationship {
	return model.UnStudiedClassStudentRelationship{
		StudentId:          stuId,
		ToBeStudiedClassId: c.Id,
		Status:             c.Status,
	}
}

// distinguishByClassType 爬虫结果只有一个切片, 根据课程类型进行放入不同的字段切片
func distinguishByClassType(classes []service.ToBeStudiedClass) service.ToBeStudiedClasses {
	var res service.ToBeStudiedClasses
	m := map[string]*[]service.ToBeStudiedClass{
		"0": &res.IdentityDevelop,
		"1": &res.CommonEducate,
		"2": &res.SpecificSkill,
	}

	for _, class := range classes {
		*m[class.Type] = append(*m[class.Type], class)
	}

	return res
}

func (c *CultivateStrategyBiz) GetCultivateStrategyFromCCNU(ctx context.Context, stuId string) (service.ToBeStudiedClasses, error) {
	cookie, err := c.cookieCli.GetCookie(ctx, stuId, new_)
	if err != nil {
		return service.ToBeStudiedClasses{}, err
	}

	req, err := http.NewRequest(http.MethodGet, courseURL, nil)
	if err != nil {
		return service.ToBeStudiedClasses{}, err
	}

	req.Header = http.Header{
		"Cookie":       []string{cookie},
		"User-Agent":   []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0"},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
	}

	resp, err := c.proxyCli.NewProxyClient(proxy.WithProxyTransport(false)).Do(req)
	if err != nil {
		return service.ToBeStudiedClasses{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	res, err := extractUnstudiedClasses(resp.Body, stuId)
	if err != nil {
		return service.ToBeStudiedClasses{}, err
	}

	return res, nil
}

func (c *CultivateStrategyBiz) GetCultivateStrategyFromDB(ctx context.Context, stuId string,
	status string, dataAlive time.Duration) (service.ToBeStudiedClasses, error) {
	relations, err := c.csData.GetClassStudentRelation(ctx, stuId, status, dataAlive)
	if err != nil {
		return service.ToBeStudiedClasses{}, err
	}

	classes := make([]service.ToBeStudiedClass, len(relations))

	// 找到然后聚合
	for i, relation := range relations {
		class, err := c.csData.GetDetailUnStudyClass(ctx, relation.ToBeStudiedClassId)
		if err != nil {
			return service.ToBeStudiedClasses{}, err
		}
		classes[i] = aggregateToService(relation, class)
	}

	res := distinguishByClassType(classes)

	return res, nil
}

// extractUnstudiedClasses 从响应提取课程
func extractUnstudiedClasses(rd io.Reader, stuId string) (service.ToBeStudiedClasses, error) {
	var (
		res       service.ToBeStudiedClasses
		classType string
		class     []string
		m         = make(map[string]struct{})
	)
	doc, err := goquery.NewDocumentFromReader(rd)
	if err != nil {
		return service.ToBeStudiedClasses{}, err
	}

	doc.Find("tr.tr-data").Each(func(i int, s *goquery.Selection) {
		if sel := s.Find("td[colspan]"); sel.Length() != 0 && isInClassTypes(sel.Text()) {
			classType = strings.TrimSpace(sel.Text())
		}

		if s.Find("td").Length() == 14 {
			class = s.Find("td").Map(func(i int, s *goquery.Selection) string {
				return strings.Join(strings.Fields(s.Text()), " ")
			})
		}

		if len(class) != 0 {
			if _, ok := m[class[1]]; !ok {
				dispatchClass(class, classType, &res, stuId)
				m[class[1]] = struct{}{}
			}
		}
	})

	return res, nil
}

// isInClassTypes 这里需要过滤一下title, 通识课下面有一堆分类, 为了简便只用大类
func isInClassTypes(c string) bool {
	for _, v := range classTypes {
		if strings.Contains(c, v) {
			return true
		}
	}
	return false
}

// dispatchClass 哪类课程放哪类切片
func dispatchClass(class []string, classType string, studiedClass *service.ToBeStudiedClasses, stuId string) {
	if len(class) == 0 {
		return
	}
	cls := classToService(class, stuId)
	m := map[string]*[]service.ToBeStudiedClass{
		identity: &studiedClass.IdentityDevelop,
		common:   &studiedClass.CommonEducate,
		specific: &studiedClass.SpecificSkill,
	}

	for tpe, classList := range m {
		if strings.Contains(classType, tpe) {
			extraModifyType(cls, tpe)
			*classList = append(*classList, *cls)
			return
		}
	}
}

func classToService(class []string, stuId string) *service.ToBeStudiedClass {
	return &service.ToBeStudiedClass{
		Id:        class[1],
		Name:      class[2],
		Status:    class[3],
		Property:  class[4],
		Credit:    class[6],
		Studiable: betterStudiable(class[13], stuId),
	}
}

// extraModifyType 用于存储在数据库, post筛选的时候有用
func extraModifyType(c *service.ToBeStudiedClass, tpe string) {
	c.Type = classMapper[tpe]
}

// betterStudiable 你师的神秘学期数字😆
func betterStudiable(s string, stuId string) string {
	// 1, 2, ... 12
	if v, ok := yearTermMapper[s]; ok {
		return v
	}

	// 2024-2025-1
	if yearTerm := strings.Split(s, "-"); len(yearTerm) == 3 {
		begin := stuId[:4]

		intYear, _ := strconv.Atoi(yearTerm[0])
		intBegin, _ := strconv.Atoi(begin)
		yearTermFinal := fmt.Sprintf("%d-%s", intYear-intBegin, yearTerm[2]) //0/1/2/3-1/2/3
		if v, ok := yearTermMapper[yearTermMapper2[yearTermFinal]]; ok {
			return v
		}
	}

	return s
}

// findClassesByStatusFromRes 在post+爬虫响应的时候使用
func findClassesByStatusFromRes(status string, res service.ToBeStudiedClasses) service.ToBeStudiedClasses {
	return service.ToBeStudiedClasses{
		IdentityDevelop: findClassesByStatus(status, res.IdentityDevelop),
		CommonEducate:   findClassesByStatus(status, res.CommonEducate),
		SpecificSkill:   findClassesByStatus(status, res.SpecificSkill),
	}
}

func findClassesByStatus(status string, classes []service.ToBeStudiedClass) []service.ToBeStudiedClass {
	var result []service.ToBeStudiedClass
	for _, class := range classes {
		if class.Status == status {
			result = append(result, class)
		}
	}

	return result
}
