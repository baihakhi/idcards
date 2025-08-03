package service

import (
	"fmt"
	"idcard/internal/model"
	"io"
	"log"

	"github.com/xuri/excelize/v2"
)

type (
	ExcelService interface {
		UpdateExcel(u *model.User) error
		ParseExcel(file io.Reader) ([][]string, error)
	}

	excelSvc struct{}
)

func NewExcelService() ExcelService {
	return &excelSvc{}
}

func (s *excelSvc) UpdateExcel(u *model.User) error {
	filePath := "./internal/data/data.xlsx"

	// Open the existing file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Printf("Error opening %s: %v\n", filePath, err)
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Println("Failed to close Excel file:", cerr)
		}
	}()

	sheet := "data"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading rows in %s: %v\n", filePath, err)
		return err
	}

	// Append to next row
	rowIndex := len(rows) + 1

	f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), u.ID)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), u.Status)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), u.NIK)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), u.Name)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), u.Phone)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), u.Address)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIndex), u.Rating)
	f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIndex), u.Notes)
	f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIndex), u.Photo)

	if err := f.Save(); err != nil {
		log.Printf("Failed to save file: %v\n", err)
		return err
	}

	return nil
}

func (s *excelSvc) ParseExcel(file io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, err
	}

	rows, err := f.GetRows("data")
	if err != nil || len(rows) < 2 {
		return nil, err
	}
	return rows, nil
}
