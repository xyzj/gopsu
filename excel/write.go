// Package excel excel功能模块
package excel

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tealeg/xlsx/v3"
)

var errSheetNotFound = func(sheetname string) error {
	if sheetname == "" {
		return nil
	}
	return errors.New("sheet " + sheetname + " not found")
}

// FileData Excel文件结构
type FileData struct {
	fileName   string
	colStyle   *xlsx.Style
	writeFile  *xlsx.File
	writeSheet *xlsx.Sheet
}

// GetCell 获取单元格，可进行合等操作
func (fd *FileData) GetCell(sheetname string, row, col int) (*xlsx.Cell, error) {
	sheet, ok := fd.writeFile.Sheet[sheetname]
	if !ok {
		return nil, errSheetNotFound((sheetname))
	}
	return sheet.Cell(row, col)
}

// GetRows 获取制定sheet的所有行
func (fd *FileData) GetRows(sheetname string) [][]string {
	sheet, ok := fd.writeFile.Sheet[sheetname]
	if !ok {
		return make([][]string, 0)
	}
	ss := make([][]string, 0, sheet.MaxRow)
	l := sheet.MaxCol
	sheet.ForEachRow(func(r *xlsx.Row) error {
		rs := make([]string, 0, l)
		r.ForEachCell(func(c *xlsx.Cell) error {
			rs = append(rs, c.Value)
			return nil
		})
		if len(rs) > 0 {
			ss = append(ss, rs)
		}
		return nil
	})
	return ss
}

// AddSheet 添加sheet
// sheetname sheet名称
func (fd *FileData) AddSheet(sheetname string) (*xlsx.Sheet, error) {
	var err error
	fd.writeSheet, err = fd.writeFile.AddSheet(sheetname)
	if err != nil {
		return nil, errors.New("excel-sheet创建失败:" + err.Error())
	}
	return fd.writeSheet, nil
}

// AddRowInSheet 在指定sheet添加行
// cells： 每个单元格的数据，任意格式
func (fd *FileData) AddRowInSheet(sheetname string, cells ...interface{}) error {
	sheet, ok := fd.writeFile.Sheet[sheetname]
	if !ok {
		return fmt.Errorf("sheet " + sheetname + " not found")
	}
	row := sheet.AddRow()
	row.SetHeight(15)
	// row.WriteSlice(cells, -1)
	for _, v := range cells {
		row.AddCell().SetValue(v)
	}
	return nil
}

// AddRow 在当前sheet添加行
// cells： 每个单元格的数据，任意格式
func (fd *FileData) AddRow(cells ...interface{}) {
	if fd.writeSheet == nil {
		fd.writeSheet, _ = fd.AddSheet("newsheet" + strconv.Itoa(len(fd.writeFile.Sheets)+1))
	}
	row := fd.writeSheet.AddRow()
	row.SetHeight(15)
	row.WriteSlice(cells, -1)
	// for _, v := range cells {
	// 	row.AddCell().SetValue(v)
	// }
}

// SetColume 设置列头
// columeName: 列头名，有多少写多少个
func (fd *FileData) SetColume(columeName ...string) {
	if fd.writeSheet == nil {
		fd.writeSheet, _ = fd.AddSheet("newsheet" + strconv.Itoa(len(fd.writeFile.Sheets)+1))
	}
	row := fd.writeSheet.AddRow()
	row.SetHeight(20)
	for _, v := range columeName {
		cell := row.AddCell()
		cell.SetStyle(fd.colStyle)
		cell.SetString(v)
	}
}

// Write 将excel数据写入到writer
// w： io.writer
func (fd *FileData) Write(w io.Writer) error {
	return fd.writeFile.Write(w)
}

// Save 保存excel数据到文件
// 返回保存的完整文件名，错误
func (fd *FileData) Save() (string, error) {
	fn := fd.fileName
	if !strings.HasSuffix(fn, ".xlsx") {
		fn += ".xlsx"
	}
	if !isExist(filepath.Dir(fn)) {
		os.MkdirAll(filepath.Dir(fn), 0o775)
	}

	err := fd.writeFile.Save(fn)
	if err != nil {
		return "", errors.New("excel-文件保存失败:" + err.Error())
	}
	return fn, nil
}

// NewExcel 创建新的excel文件
// filename: 需要保存的文件路径，可不加扩展名
// 返回：excel数据格式，错误
func NewExcel(filename string) (*FileData, error) {
	var err error
	fd := &FileData{
		writeFile: xlsx.NewFile(),
		colStyle:  xlsx.NewStyle(),
	}
	fd.colStyle.Alignment.Horizontal = "center"
	fd.colStyle.Font.Bold = true
	fd.colStyle.ApplyAlignment = true
	fd.colStyle.ApplyFont = true
	// fd.writeSheet, err = fd.writeFile.AddSheet(time.Now().Format("2006-01-02"))
	// if err != nil {
	// 	return nil, errors.New("excel-sheet创建失败:" + err.Error())
	// }
	fd.fileName = filename
	return fd, err
}

// NewExcelFromBinary 从文件读取xlsx
func NewExcelFromBinary(bs []byte, filename string) (*FileData, error) {
	var err error
	var fd *FileData
	xf, err := xlsx.OpenBinary(bs)
	if err != nil {
		fd = &FileData{
			writeFile: xlsx.NewFile(),
			colStyle:  xlsx.NewStyle(),
		}
	} else {
		fd = &FileData{
			writeFile: xf,
			colStyle:  xlsx.NewStyle(),
		}
	}
	fd.colStyle.Alignment.Horizontal = "center"
	fd.colStyle.Font.Bold = true
	fd.colStyle.ApplyAlignment = true
	fd.colStyle.ApplyFont = true
	// fd.writeSheet, err = fd.writeFile.AddSheet(time.Now().Format("2006-01-02"))
	// if err != nil {
	// 	return nil, errors.New("excel-sheet创建失败:" + err.Error())
	// }
	fd.fileName = filename
	return fd, err
}

func NewExcelFromUpload(file multipart.File, filename string) (*FileData, error) {
	fb, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return NewExcelFromBinary(fb, filename)
}

func isExist(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}
