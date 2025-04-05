package reptile

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/dubbogo/net/html"
	"net/http"
	"strings"
	"time"
)

// 定义日历信息结构体
type CalendarInfo struct {
	Link       string
	Year       string // 以学年的开始作为标记
	PDFLink    string
	ImageLinks []string
}

// 不是很成功的设计但是方便
const BASEURL = "https://jwc.ccnu.edu.cn"

// 定义爬虫接口
type Reptile interface {
	GetCalendarLink() ([]CalendarInfo, error)
	FetchPDFOrImageLinksFromPage(url string) (string, []string, error)
}

// 定义结构体
type reptile struct{}

// 创建新爬虫实例
func NewReptile() Reptile {
	return &reptile{}
}

// 创建 HTTP 客户端
var client = &http.Client{
	Timeout: 10 * time.Second, // 设置超时
}

// User-Agent 头信息
var headers = map[string]string{
	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
}

// GetCalendarLink 获取校历的链接

func (r *reptile) GetCalendarLink() ([]CalendarInfo, error) {
	url := "https://jwc.ccnu.edu.cn/index/hdxl.htm"

	// 创建 HTTP 请求，添加 User-Agent 头
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var calendarInfos []CalendarInfo

	// 解析 <li> 结构
	doc.Find("ul.list li").Each(func(i int, s *goquery.Selection) {
		aTag := s.Find("a")

		// 提取链接
		link, exists := aTag.Attr("href")
		if !exists {
			return
		}
		// 处理相对路径
		if strings.HasPrefix(link, "../") {
			link = BASEURL + link[2:]
		} else {
			link = BASEURL + link
		}

		// 提取学年
		year := strings.TrimSpace(aTag.Text())
		if len(year) >= 4 {
			year = year[:4]
		}

		// 存储数据
		calendarInfos = append(calendarInfos, CalendarInfo{
			Link: link,
			Year: year,
		})
	})

	return calendarInfos, nil
}

// FetchPDFOrImageLinksFromPage 获取 PDF 和图片链接
func (r *reptile) FetchPDFOrImageLinksFromPage(url string) (string, []string, error) {
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	// 解析 HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", nil, err
	}

	// 变量初始化
	var pdfLink string
	var imageLinks []string

	// 递归查找
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// 找到 PDF 文件
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" && strings.HasSuffix(attr.Val, ".pdf") {
						pdfLink = BASEURL + attr.Val
					}
				}
			}

			// 找到 <div class="v_news_content">
			if n.Type == html.ElementNode && n.Data == "div" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && attr.Val == "v_news_content" {
						findImages(n, &imageLinks)
					}
				}
			}
		}

		// 递归遍历子节点
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// 开始遍历
	traverse(doc)
	return pdfLink, imageLinks, nil
}

// findImages 查找所有 <img> 标签
func findImages(n *html.Node, imageLinks *[]string) {
	if n.Type == html.ElementNode && n.Data == "img" {
		for _, attr := range n.Attr {
			if attr.Key == "src" {
				*imageLinks = append(*imageLinks, BASEURL+attr.Val)
			}
		}
	}
	// 递归查找
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findImages(c, imageLinks)
	}
}
