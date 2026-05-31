package apiresp

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/leo/iop/server/internal/shared/errors"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// XLSX writes rows as a simple first-sheet Excel attachment. Keep this helper
// deliberately small: import/export callers own column order and validation.
func XLSX(c *gin.Context, filename string, rows [][]string) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for r, row := range rows {
		for col, val := range row {
			cell, _ := excelize.CoordinatesToCellName(col+1, r+1)
			_ = f.SetCellStr(sheet, cell, val)
		}
	}
	var buf bytes.Buffer
	_ = f.Write(&buf)
	_ = f.Close()
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, xlsxContentType, buf.Bytes())
}

// ParseTabularUpload accepts .xlsx and .csv uploads and returns a matrix of cell
// strings. It preserves the existing CSV path while allowing Excel templates for
// modules that need proper spreadsheet import/export.
func ParseTabularUpload(c *gin.Context, field string) ([][]string, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "sheet.no_file", "请上传 Excel 或 CSV 文件", err)
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".xlsx" {
		return readXLSXFile(fileHeader)
	}
	if ext == ".csv" || ext == ".txt" || ext == "" {
		return readCSVFile(fileHeader)
	}
	return nil, errors.New(errors.KindParam, "sheet.unsupported_type", "仅支持 .xlsx 或 .csv 文件")
}

func readXLSXFile(fh *multipart.FileHeader) ([][]string, error) {
	r, err := fh.Open()
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "sheet.open_failed", "读取上传文件失败", err)
	}
	defer r.Close()
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "sheet.parse_failed", "Excel 解析失败，请检查格式", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, errors.New(errors.KindParam, "sheet.no_sheet", "Excel 文件没有工作表")
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, errors.Wrap(errors.KindParam, "sheet.read_failed", "读取 Excel 工作表失败", err)
	}
	return rows, nil
}
