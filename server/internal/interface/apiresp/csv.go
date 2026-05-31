package apiresp

import (
	"bytes"
	"encoding/csv"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/shared/errors"
)

// utf8BOM is prepended to CSV downloads so Excel (esp. on Windows) detects UTF-8
// and renders CJK text correctly instead of mojibake.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// CSV writes rows as a UTF-8 CSV attachment with a BOM. filename is the suggested
// download name (e.g. "departments.csv").
func CSV(c *gin.Context, filename string, rows [][]string) {
	var buf bytes.Buffer
	buf.Write(utf8BOM)
	w := csv.NewWriter(&buf)
	_ = w.WriteAll(rows)
	w.Flush()
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// ParseCSVUpload reads the first uploaded file field ("file") of a multipart form
// and returns its parsed CSV records. A leading UTF-8 BOM is stripped. Returns a
// KindParam error if no file is present or the CSV is malformed.
func ParseCSVUpload(c *gin.Context, field string) ([][]string, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "csv.no_file", "请上传 CSV 文件", err)
	}
	return readCSVFile(fileHeader)
}

func readCSVFile(fh *multipart.FileHeader) ([][]string, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "csv.open_failed", "读取上传文件失败", err)
	}
	defer f.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(f); err != nil {
		return nil, errors.Wrap(errors.KindParam, "csv.read_failed", "读取上传文件失败", err)
	}
	data := bytes.TrimPrefix(buf.Bytes(), utf8BOM)
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // tolerate ragged rows; handlers default missing cols
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "csv.parse_failed", "CSV 解析失败，请检查格式", err)
	}
	return records, nil
}
