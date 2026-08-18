package eventparticipants

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

const (
	headerFullName   = "full_name"
	headerBirthDate  = "birth_date"
	headerUnionCard  = "union_card_number"
	headerSKS        = "sks_barcode"
	headerDirection  = "direction"
)

// ParseImportFile преобразует CSV/XLSX в единый набор строк. Бизнес-валидация
// выполняется отдельно, чтобы каждая ошибочная строка попала в ImportResult.
func ParseImportFile(filename string, reader io.Reader) ([]ImportRecord, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read import: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".xlsx" || (ext == "" && len(data) >= 2 && data[0] == 'P' && data[1] == 'K') {
		return parseXLSX(bytes.NewReader(data))
	}
	if ext != "" && ext != ".csv" && ext != ".txt" {
		return nil, fmt.Errorf("%w: поддерживаются CSV и XLSX", ErrValidation)
	}
	return parseCSV(data)
}

func parseCSV(data []byte) ([]ImportRecord, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = false
	reader.Comma = detectDelimiter(data)

	rows := make([][]string, 0)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: некорректный CSV: %v", ErrValidation, err)
		}
		rows = append(rows, row)
	}
	return recordsFromRows(rows)
}

func detectDelimiter(data []byte) rune {
	firstLine := data
	if newline := bytes.IndexByte(data, '\n'); newline >= 0 {
		firstLine = data[:newline]
	}
	candidates := []rune{',', ';', '\t'}
	best := ','
	bestCount := -1
	for _, candidate := range candidates {
		count := bytes.Count(firstLine, []byte(string(candidate)))
		if count > bestCount {
			best, bestCount = candidate, count
		}
	}
	return best
}

func parseXLSX(reader io.Reader) ([]ImportRecord, error) {
	book, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: некорректный XLSX", ErrValidation)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("%w: XLSX не содержит листов", ErrValidation)
	}
	rows, err := book.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("%w: не удалось прочитать XLSX", ErrValidation)
	}
	return recordsFromRows(rows)
}

func recordsFromRows(rows [][]string) ([]ImportRecord, error) {
	headerRow := -1
	for i, row := range rows {
		if !emptyRow(row) {
			headerRow = i
			break
		}
	}
	if headerRow < 0 {
		return nil, fmt.Errorf("%w: файл пуст", ErrValidation)
	}
	columns := mapHeaders(rows[headerRow])
	if _, ok := columns[headerFullName]; !ok {
		return nil, fmt.Errorf("%w: отсутствует колонка full_name/ФИО", ErrValidation)
	}
	if _, ok := columns[headerBirthDate]; !ok {
		return nil, fmt.Errorf("%w: отсутствует колонка birth_date/Дата рождения", ErrValidation)
	}

	records := make([]ImportRecord, 0, len(rows)-headerRow-1)
	for i := headerRow + 1; i < len(rows); i++ {
		if emptyRow(rows[i]) {
			continue
		}
		records = append(records, ImportRecord{
			Line:            i + 1,
			FullName:        valueAt(rows[i], columns[headerFullName]),
			BirthDate:       valueAt(rows[i], columns[headerBirthDate]),
			UnionCardNumber: valueForHeader(rows[i], columns, headerUnionCard),
			SKSBarcode:      valueForHeader(rows[i], columns, headerSKS),
			Direction:       valueForHeader(rows[i], columns, headerDirection),
		})
	}
	return records, nil
}

func mapHeaders(row []string) map[string]int {
	columns := make(map[string]int)
	for index, raw := range row {
		switch normalizeHeader(raw) {
		case "full name", "fio", "фио", "ф и о", "полное имя":
			columns[headerFullName] = index
		case "birth date", "date of birth", "дата рождения", "дата рожд":
			columns[headerBirthDate] = index
		case "union card number", "union card", "номер профсоюзного билета", "номер профбилета", "профбилет":
			columns[headerUnionCard] = index
		case "sks barcode", "barcode", "barcode скс", "штрихкод скс":
			columns[headerSKS] = index
		case "direction", "track", "направление", "направления", "трек":
			columns[headerDirection] = index
		}
	}
	return columns
}

func normalizeHeader(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "\uFEFF")
	value = strings.ToLower(value)
	value = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func valueForHeader(row []string, columns map[string]int, header string) string {
	index, ok := columns[header]
	if !ok {
		return ""
	}
	return valueAt(row, index)
}

func valueAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	value := strings.TrimSpace(row[index])
	if !utf8.ValidString(value) {
		return ""
	}
	return value
}

func emptyRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func parseBirthDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"2006-01-02", "02.01.2006", "02/01/2006", time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	if serial, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64); err == nil && serial > 0 {
		if parsed, err := excelize.ExcelDateToTime(serial, false); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, ErrValidation
}

func exportCSV(participants []Participant) (*ExportFile, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"full_name", "birth_date", "union_card_number", "sks_barcode", "direction", "status"}); err != nil {
		return nil, err
	}
	for i := range participants {
		if err := writer.Write(exportRow(&participants[i])); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return &ExportFile{Name: "event-participants.csv", ContentType: "text/csv; charset=utf-8", Data: buffer.Bytes()}, nil
}

func exportXLSX(participants []Participant) (*ExportFile, error) {
	book := excelize.NewFile()
	defer book.Close()
	sheet := book.GetSheetName(0)
	headers := []string{"full_name", "birth_date", "union_card_number", "sks_barcode", "direction", "status"}
	for column, value := range headers {
		cell, _ := excelize.CoordinatesToCellName(column+1, 1)
		if err := book.SetCellValue(sheet, cell, value); err != nil {
			return nil, err
		}
	}
	for rowIndex := range participants {
		for column, value := range exportRow(&participants[rowIndex]) {
			cell, _ := excelize.CoordinatesToCellName(column+1, rowIndex+2)
			if err := book.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return &ExportFile{
		Name:        "event-participants.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Data:        buffer.Bytes(),
	}, nil
}

func exportRow(participant *Participant) []string {
	return []string{
		participant.FullName, participant.BirthDate.Format("2006-01-02"),
		optionalValue(participant.UnionCardNumber), optionalValue(participant.SKSBarcode),
		optionalValue(participant.DirectionName), participant.Status,
	}
}

func optionalValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
