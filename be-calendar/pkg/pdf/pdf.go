package pdf

import (
	"bytes"
	"fmt"
	"github.com/jung-kurt/gofpdf"
	"io"
	"net/http"
	"path"
	"strings"
)

// CreatePDFfromImageLinks 从给定的图片链接数组生成 PDF，并返回其字节流
func CreatePDFfromImageLinks(imageLinks []string) ([]byte, error) {
	// 防止出现过大字节流压垮服务器
	if len(imageLinks) > 10 {
		return nil, fmt.Errorf("too many images, limit is 10")
	}

	// 创建一个新的 PDF 实例，默认页面尺寸
	pdf := gofpdf.New("P", "cm", "", "")

	// 设置页面宽度为16.5cm
	pageWidth := 16.5

	// 遍历所有链接，并将图片添加到 PDF 中
	for _, link := range imageLinks {
		// 获取图片字节流
		imgBytes, err := GetBytesFromLink(link)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %v", err)
		}

		// 确定图片类型
		imgType := DetermineImageType(link, imgBytes)

		// 添加图片到 PDF
		addImageToPDF(pdf, imgBytes, imgType, pageWidth)
	}

	// 生成 PDF 并返回字节流
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %v", err)
	}

	return buf.Bytes(), nil
}

// DetermineImageType 通过链接后缀或图片内容确定图片类型
func DetermineImageType(link string, imgBytes []byte) string {
	// 根据 URL 后缀判断类型
	ext := strings.ToLower(path.Ext(link))
	switch ext {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".gif":
		return "gif"
	}

	// 如果无法通过后缀判断，尝试通过内容判断
	contentType := http.DetectContentType(imgBytes)
	switch contentType {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	default:
		return "png" // 默认返回 PNG 类型
	}
}

// GetBytesFromLink 从给定的链接获取图片的字节流
func GetBytesFromLink(url string) ([]byte, error) {

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return imgBytes, nil
}

// addImageToPDF 添加图片并确保它的宽度为 pageWidth，高度自适应
func addImageToPDF(pdf *gofpdf.Fpdf, imgBytes []byte, imgType string, pageWidth float64) {
	// 获取图片的尺寸
	imgOpt := gofpdf.ImageOptions{
		ImageType:             imgType,
		ReadDpi:               false,
		AllowNegativePosition: false,
	}

	info := pdf.RegisterImageOptionsReader("image", imgOpt, bytes.NewReader(imgBytes))

	// 计算图片高度，使其与页面宽度适配
	imgWidth := info.Width()
	imgHeight := info.Height()
	scale := pageWidth / imgWidth
	pageHeight := imgHeight * scale

	// 添加一个自定义大小的页面
	pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pageWidth, Ht: pageHeight})

	// 在页面上绘制图片
	pdf.ImageOptions("image", 0, 0, pageWidth, pageHeight, false, imgOpt, 0, "")
}
