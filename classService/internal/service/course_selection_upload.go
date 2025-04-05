package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/asynccnu/ccnubox-be/classService/internal/model"
	"net/http"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 必要的列的索引
type NecessaryIndex struct {
	ClassTimeIdx  uint `json:"class_time_idx"`
	ClassWhereIdx uint `json:"class_where_idx"`
}

type UploadReq struct {
	Year     string                    `json:"year"`
	Semester string                    `json:"semester"`
	Sheets   map[string]NecessaryIndex `json:"sheets"` // sheet名，以及每个sheet的上课时间和教学地点的索引[数字],索引从0开始,比如上课时间是第7列，就传6
}

type FreeClassRoomSaver interface {
	SaveFreeClassRoomInfo(ctx context.Context, year, semester string, cwtPairs []model.CTWPair) error
}

// 处理上传选课手册的http服务
type SelectionUploader struct {
	freeClassRoom FreeClassRoomSaver
}

func NewSelectionUploader(freeClassRoom FreeClassRoomSaver) *SelectionUploader {
	return &SelectionUploader{
		freeClassRoom: freeClassRoom,
	}
}
func (s *SelectionUploader) UploadSelection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB 内存，超出的部分存到磁盘
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// 解码 JSON
	jsonData := r.FormValue("json_data") // 对应前端字段名
	var req UploadReq
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 解析上传的文件
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	f, err := excelize.OpenReader(buf)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ctwPairs, err := getCWTPairs(f, req.Sheets)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to handle excel:%v", err), http.StatusInternalServerError)
	}

	err = s.freeClassRoom.SaveFreeClassRoomInfo(r.Context(), req.Year, req.Semester, ctwPairs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save free class room info:%v", err), http.StatusInternalServerError)
	}

	// 设置响应头，内容类型为 JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(`{"msg":"success"}`))
}

func getCWTPairs(f *excelize.File, mp map[string]NecessaryIndex) ([]model.CTWPair, error) {
	type ClassRoomTimeData struct {
		Time  string
		Where string
	}
	var datas []ClassRoomTimeData

	//读取每个sheets的上课时间和上课地点
	for sheetName, necessaryIndex := range mp {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(rows); i++ {
			if i == 0 {
				continue
			}
			datas = append(datas, ClassRoomTimeData{
				Time:  rows[i][necessaryIndex.ClassTimeIdx],
				Where: rows[i][necessaryIndex.ClassWhereIdx],
			})
		}
	}

	var ctwPairs []model.CTWPair

	for _, data := range datas {
		ctimes := parseTime(data.Time)
		wheres := strings.Split(data.Where, ";")

		for _, ct := range ctimes {
			for _, where := range wheres {
				ctwPairs = append(ctwPairs, model.CTWPair{
					CT:    ct,
					Where: where,
				})
			}
		}
	}
	return ctwPairs, nil
}

// 看几种典型的时间格式
// 星期四第3-4节{4-19周}
// 星期一第1-2节{4-18周(双)};星期二第7-8节{4-19周}
// 星期一第5-8节{4-6周(双),7-8周};星期二第5-8节{4-6周(双),7-8周};星期四第1-4节{4-6周(双),7-8周};星期五第1-4节{4-6周(双),7-8周}
// 星期一第9-10节{5-17周(单)};星期二第1-2节{4-19周}
func parseTime(val string) []model.CTime {
	var mp = map[string]int{
		"一": 1,
		"二": 2,
		"三": 3,
		"四": 4,
		"五": 5,
		"六": 6,
		"日": 7,
	}

	uniteTimes := strings.Split(val, ";")
	res := make([]model.CTime, 0, len(uniteTimes))
	for _, uniteTime := range uniteTimes {
		var tt model.CTime

		index := strings.Index(uniteTime, "{")
		tmp1 := uniteTime[:index]                     //代表 "星期一第1-2节" 这样的部分
		tmp2 := uniteTime[index+1 : len(uniteTime)-1] //代表 4-19周 这个部分

		//获取星期几和第几节
		index = strings.Index(tmp1, "第")
		xinqi := tmp1[:index]  // 代表如 "星期一"的部分
		jieshu := tmp1[index:] // 代表如 "第1-2节" 的部分

		dayStr := strings.TrimPrefix(xinqi, "星期")
		day := mp[dayStr] //星期几

		var jieStart, jieEnd int
		fmt.Sscanf(jieshu, "第%d-%d节", &jieStart, &jieEnd) //第几节

		tt.Day = day
		for i := jieStart; i <= jieEnd; i++ {
			tt.Sections = append(tt.Sections, i)
		}

		//开始获取周数
		weekStrs := strings.Split(tmp2, ",")

		for _, weekStr := range weekStrs {

			index = strings.Index(weekStr, "(")
			var weekStart, weekEnd int
			var pattern int // 1代表单周，2代表双周，0代表没有
			//先看看有没有括号
			if index == -1 {
				fmt.Sscanf(weekStr, "%d-%d周", &weekStart, &weekEnd)
			} else {
				var patternStr string
				//"%s" 读取的是一个不包含空格的字符串，它会直接匹配 "双)"
				//括号 () 仍然被解析为字符串的一部分，因为 Sscanf 不能自动去掉这些符号
				fmt.Sscanf(weekStr, "%d-%d周(%s)", &weekStart, &weekEnd, &patternStr)
				// 手动去掉末尾的 ")"
				patternStr = strings.TrimSuffix(patternStr, ")")
				if patternStr == "单" {
					pattern = 1
				}
				if patternStr == "双" {
					pattern = 2
				}
			}
			for i := weekStart; i <= weekEnd; i++ {
				if pattern == 0 {
					tt.Weeks = append(tt.Weeks, i)
				}
				if pattern == 1 {
					if i%2 == 1 {
						tt.Weeks = append(tt.Weeks, i)
					}
				}
				if pattern == 2 {
					if i%2 == 0 {
						tt.Weeks = append(tt.Weeks, i)
					}
				}
			}
		}
		res = append(res, tt)
		//fmt.Printf("%+v\n",res)
	}
	return res
}
