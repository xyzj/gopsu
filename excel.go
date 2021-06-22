package gopsu

import (
	"fmt"

	"github.com/tealeg/xlsx"
)

type excelData struct {
	fileName  string
	colStyle  *xlsx.Style
	xlsxFile  *xlsx.File
	xlsxSheet *xlsx.Sheet
}

// AddSheet 添加sheet
// sheetname sheet名称
func (e *excelData) AddSheet(sheetname string) (*xlsx.Sheet, error) {
	var err error
	e.xlsxSheet, err = e.xlsxFile.AddSheet(sheetname)
	if err != nil {
		return nil, fmt.Errorf("excel-sheet创建失败:" + err.Error())
	}
	return e.xlsxSheet, nil
}

// AddRowInSheet 在指定sheet添加行
// cells： 每个单元格的数据，任意格式
func (e *excelData) AddRowInSheet(sheetname string, cells ...interface{}) {
	sheet := e.xlsxFile.Sheet[sheetname]
	row := sheet.AddRow()
	row.SetHeight(15)
	// row.WriteSlice(cells, -1)
	for _, v := range cells {
		row.AddCell().SetValue(v)
	}
}

// AddRow 在当前sheet添加行
// cells： 每个单元格的数据，任意格式
func (e *excelData) AddRow(cells ...interface{}) {
	if e.xlsxSheet == nil {
		e.xlsxSheet, _ = e.AddSheet(fmt.Sprintf("newsheet%d", len(e.xlsxFile.Sheets)+1))
	}
	row := e.xlsxSheet.AddRow()
	row.SetHeight(15)
	// row.WriteSlice(cells, -1)
	for _, v := range cells {
		row.AddCell().SetValue(v)
	}
}

// SetColume 设置列头
// columeName: 列头名，有多少写多少个
func (e *excelData) SetColume(columeName ...string) {
	if e.xlsxSheet == nil {
		e.xlsxSheet, _ = e.AddSheet(fmt.Sprintf("newsheet%d", len(e.xlsxFile.Sheets)+1))
	}
	row := e.xlsxSheet.AddRow()
	row.SetHeight(20)
	for _, v := range columeName {
		cell := row.AddCell()
		cell.SetStyle(e.colStyle)
		cell.SetString(v)
	}
}

// Save 保存excel数据到文件
// 返回保存的完整文件名，错误
func (e *excelData) Save() (string, error) {
	err := e.xlsxFile.Save(e.fileName)
	if err != nil {
		return "", fmt.Errorf("excel-文件保存失败:" + err.Error())
	} else {
		return e.fileName, nil
	}
}

// NewExcel 创建新的excel文件
// filename: 需要保存的文件名头，如："事件日志"，不要加扩展名
// 返回：excel数据格式，错误
func NewExcel(filename string) (*excelData, error) {
	var err error
	e := &excelData{
		xlsxFile: xlsx.NewFile(),
		colStyle: xlsx.NewStyle(),
	}
	e.colStyle.Alignment.Horizontal = "center"
	e.colStyle.Font.Bold = true
	e.colStyle.ApplyAlignment = true
	e.colStyle.ApplyFont = true
	// e.xlsxSheet, err = e.xlsxFile.AddSheet(time.Now().Format("2006-01-02"))
	// if err != nil {
	// 	return nil, fmt.Errorf("excel-sheet创建失败:" + err.Error())
	// }
	e.fileName = filename
	return e, err
}
