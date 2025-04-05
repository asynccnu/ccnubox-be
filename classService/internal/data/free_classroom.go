package data

import (
	"context"
	"fmt"
	clog "github.com/asynccnu/ccnubox-be/classService/internal/log"
	"github.com/asynccnu/ccnubox-be/classService/internal/model"
	"github.com/olivere/elastic/v7"
)

const (
	freeClassroomIndex   = "free_classroom"
	freeClassroomMapping = `{
	"mappings": {
		"properties": {
			"year": { "type": "keyword" },
			"semester": { "type": "keyword" },
			"where": { "type": "keyword" },
			"weeks": { "type": "integer" },
			"day": { "type": "integer" },
			"sections": { "type": "integer" }
		}
	}
}`
)

type FreeClassroomData struct {
	cli *elastic.Client
}

func NewFreeClassroomData(cli *elastic.Client) *FreeClassroomData {
	return &FreeClassroomData{
		cli: cli,
	}
}
func (f *FreeClassroomData) GetAllClassroom(ctx context.Context, year, semester, wherePrefix string) ([]string, error) {
	return f.getAllWheres(ctx, year, semester, wherePrefix)
}

func (f *FreeClassroomData) AddClassroomOccupancy(ctx context.Context, year, semester string, cwtPairs ...model.CTWPair) error {
	//type ClassroomOccupancy struct {
	//	Year     string `json:"year"`
	//	Semester string `json:"semester"`
	//	Where    string `json:"where"`
	//	Weeks    []int  `json:"weeks"`
	//	Day      int    `json:"day"`
	//	Sections []int  `json:"sections"`
	//}
	//co := ClassroomOccupancy{
	//	Year:     year,
	//	Semester: semester,
	//	Where:    cwtPair.Where,
	//	Weeks:    cwtPair.CT.Weeks,
	//	Day:      cwtPair.CT.Day,
	//	Sections: cwtPair.CT.Sections,
	//}
	//_, err := f.cli.Index().
	//	Index(freeClassroomIndex).
	//	Id(fmt.Sprintf("%v-%v-%v", year, semester, cwtPair)).
	//	BodyJson(co).
	//	Refresh("true").
	//	Do(ctx)
	//if err != nil {
	//	clog.LogPrinter.Errorf("es: failed to add classroom_occupancy[%+v]: %v", co, err)
	//	return err
	//}
	//return nil
	// 定义文档结构
	type ClassroomOccupancy struct {
		Year     string `json:"year"`
		Semester string `json:"semester"`
		Where    string `json:"where"`
		Weeks    []int  `json:"weeks"`
		Day      int    `json:"day"`
		Sections []int  `json:"sections"`
	}

	// 检查空数据
	if len(cwtPairs) == 0 {
		return nil
	}

	// 创建批量请求
	bulkRequest := f.cli.Bulk()

	for _, cwtPair := range cwtPairs {
		// 构建文档
		doc := ClassroomOccupancy{
			Year:     year,
			Semester: semester,
			Where:    cwtPair.Where,
			Weeks:    cwtPair.CT.Weeks,
			Day:      cwtPair.CT.Day,
			Sections: cwtPair.CT.Sections,
		}

		// 生成唯一ID
		docID := fmt.Sprintf("%s-%s-%s-%d-%v-%v",
			year,
			semester,
			cwtPair.Where,
			cwtPair.CT.Day,
			cwtPair.CT.Weeks,
			cwtPair.CT.Sections)

		// 添加到批量请求
		req := elastic.NewBulkIndexRequest().
			Index(freeClassroomIndex).
			Id(docID).
			Doc(doc)
		bulkRequest = bulkRequest.Add(req)
	}

	// 执行批量操作
	bulkResponse, err := bulkRequest.Do(ctx)
	if err != nil {
		clog.LogPrinter.Errorf("es: failed to bulk add %d classroom_occupancy records: %v", len(cwtPairs), err)
		return fmt.Errorf("failed to bulk add records: %w", err)
	}

	// 检查失败的文档
	if bulkResponse.Errors {
		errorCount := 0
		for _, failed := range bulkResponse.Failed() {
			errorCount++
			clog.LogPrinter.Errorf("es: failed to index classroom_occupancy[%s]: %s", failed.Id, failed.Error)
		}
		return fmt.Errorf("%d out of %d records failed to index", errorCount, len(cwtPairs))
	}

	clog.LogPrinter.Infof("successfully indexed %d classroom_occupancy records", len(cwtPairs))
	return nil
}

// ClearClassroomOccupancy 删除教室占用信息，只保留year和semester的
func (f *FreeClassroomData) ClearClassroomOccupancy(ctx context.Context, year, semester string) error {
	query := elastic.NewBoolQuery().
		Should(
			elastic.NewBoolQuery().MustNot(elastic.NewTermQuery("year", year)),
			elastic.NewBoolQuery().MustNot(elastic.NewTermQuery("semester", semester)),
		)

	deleteResponse, err := f.cli.DeleteByQuery().
		Index(freeClassroomIndex).
		Query(query).
		Conflicts("proceed"). // 忽略冲突
		Slices("auto").
		Do(ctx)

	if err != nil {
		clog.LogPrinter.Errorf("delete classroom occupancy failed: %v", err)
		return err
	}

	// 二次验证是否全部删除
	if deleteResponse.Deleted > 0 {
		count, _ := f.cli.Count().Index(freeClassroomIndex).Do(ctx)
		if count > 0 {
			clog.LogPrinter.Warnf("after deleted,still have %d classroom occupancy", count)
		}
	}

	clog.LogPrinter.Infof("delete %d classroom occupancy successfully", deleteResponse.Deleted)
	return nil
}

func (f *FreeClassroomData) QueryAvailableClassrooms(ctx context.Context, year, semester string, week, day, section int, wherePrefix string) (map[string]bool, error) {

	allWheres, err := f.getAllWheres(ctx, year, semester, wherePrefix)
	if err != nil {
		return nil, err
	}
	var occupancyStat = make(map[string]bool, len(allWheres))
	//先全部标记为空闲
	for _, w := range allWheres {
		occupancyStat[w] = true
	}

	occupiedWheres, err := f.getOccupiedWheres(ctx, year, semester, week, day, section, wherePrefix)
	if err != nil {
		return nil, err
	}
	//标记为占用
	for _, w := range occupiedWheres {
		occupancyStat[w] = false
	}

	return occupancyStat, nil
}
func (f *FreeClassroomData) getAllWheres(ctx context.Context, year, semester, wherePrefix string) ([]string, error) {
	boolQuery := elastic.NewBoolQuery().
		Must(
			elastic.NewTermQuery("year", year),
			elastic.NewTermQuery("semester", semester),
			elastic.NewPrefixQuery("where", wherePrefix),
		)
	termsAgg := elastic.NewTermsAggregation().Field("where").Size(10000)

	//只关心聚合结果，不需要文档内容 size设置为0
	searchResult, err := f.cli.Search().
		Index(freeClassroomIndex).
		Query(boolQuery).
		Aggregation("unique_wheres", termsAgg).
		Size(0).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	aggResult, found := searchResult.Aggregations.Terms("unique_wheres")
	if !found {
		return []string{}, nil
	}

	var wheres []string
	for _, bucket := range aggResult.Buckets {
		wheres = append(wheres, bucket.Key.(string))
	}

	return wheres, nil
}

func (f *FreeClassroomData) getOccupiedWheres(ctx context.Context, year, semester string, week, day, section int, wherePrefix string) ([]string, error) {
	boolQuery := elastic.NewBoolQuery().
		Must(
			elastic.NewTermQuery("year", year),
			elastic.NewTermQuery("semester", semester),
			elastic.NewPrefixQuery("where", wherePrefix),
			elastic.NewTermQuery("weeks", week),
			elastic.NewTermQuery("day", day),
			elastic.NewTermQuery("sections", section),
		)
	termsAgg := elastic.NewTermsAggregation().Field("where").Size(10000)

	//只关心聚合结果，不需要文档内容
	searchResult, err := f.cli.Search().
		Index(freeClassroomIndex).
		Query(boolQuery).
		Aggregation("occupied_wheres", termsAgg).
		Size(0).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	aggResult, found := searchResult.Aggregations.Terms("occupied_wheres")
	if !found {
		return []string{}, nil
	}

	var wheres []string
	for _, bucket := range aggResult.Buckets {
		wheres = append(wheres, bucket.Key.(string))
	}

	return wheres, nil
}

//func difference(all []string, occupied []string) []string {
//	occupiedSet := make(map[string]struct{})
//	for _, w := range occupied {
//		occupiedSet[w] = struct{}{}
//	}
//
//	var free []string
//	for _, w := range all {
//		if _, ok := occupiedSet[w]; !ok {
//			free = append(free, w)
//		}
//	}
//	return free
//}
