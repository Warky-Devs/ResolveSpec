package security

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/bitechdev/ResolveSpec/pkg/logger"
	"github.com/bitechdev/ResolveSpec/pkg/reflection"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ColumnSecurity struct {
	Schema       string
	Tablename    string
	Path         []string
	ExtraFilters map[string]string
	UserID       int
	Accesstype   string `json:"accesstype"`
	MaskStart    int
	MaskEnd      int
	MaskInvert   bool
	MaskChar     string
	Control      string `json:"control"`
	ID           int    `json:"id"`
}

type RowSecurity struct {
	Schema    string
	Tablename string
	Template  string
	HasBlock  bool
	UserID    int
}

func (m *RowSecurity) GetTemplate(pPrimaryKeyName string, pModelType reflect.Type) string {
	str := m.Template
	str = strings.ReplaceAll(str, "{PrimaryKeyName}", pPrimaryKeyName)
	str = strings.ReplaceAll(str, "{TableName}", m.Tablename)
	str = strings.ReplaceAll(str, "{SchemaName}", m.Schema)
	str = strings.ReplaceAll(str, "{UserID}", fmt.Sprintf("%d", m.UserID))
	return str
}

// Callback function types for customizing security behavior
type (
	// AuthenticateFunc extracts user ID and roles from HTTP request
	// Return userID, roles, error. If error is not nil, request will be rejected.
	AuthenticateFunc func(r *http.Request) (userID int, roles string, err error)

	// LoadColumnSecurityFunc loads column security rules for a user and entity
	// Override this to customize how column security is loaded from your data source
	LoadColumnSecurityFunc func(pUserID int, pSchema, pTablename string) ([]ColumnSecurity, error)

	// LoadRowSecurityFunc loads row security rules for a user and entity
	// Override this to customize how row security is loaded from your data source
	LoadRowSecurityFunc func(pUserID int, pSchema, pTablename string) (RowSecurity, error)
)

type SecurityList struct {
	ColumnSecurityMutex sync.RWMutex
	ColumnSecurity      map[string][]ColumnSecurity
	RowSecurityMutex    sync.RWMutex
	RowSecurity         map[string]RowSecurity

	// Overridable callbacks
	AuthenticateCallback       AuthenticateFunc
	LoadColumnSecurityCallback LoadColumnSecurityFunc
	LoadRowSecurityCallback    LoadRowSecurityFunc
}

const SECURITY_CONTEXT_KEY = "SecurityList"

var GlobalSecurity SecurityList

// SetSecurityMiddleware adds security context to requests
func SetSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), SECURITY_CONTEXT_KEY, &GlobalSecurity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func maskString(pString string, maskStart, maskEnd int, maskChar string, invert bool) string {
	strLen := len(pString)
	middleIndex := (strLen / 2)
	newStr := ""
	if maskStart == 0 && maskEnd == 0 {
		maskStart = strLen
		maskEnd = strLen
	}
	if maskEnd > strLen {
		maskEnd = strLen
	}
	if maskStart > strLen {
		maskStart = strLen
	}
	if maskChar == "" {
		maskChar = "*"
	}
	for index, char := range pString {
		if invert && index >= middleIndex-maskStart && index <= middleIndex {
			newStr = newStr + maskChar
			continue
		}
		if invert && index <= middleIndex+maskEnd && index >= middleIndex {
			newStr = newStr + maskChar
			continue
		}
		if !invert && index <= maskStart {
			newStr = newStr + maskChar
			continue
		}
		if !invert && index >= strLen-1-maskEnd {
			newStr = newStr + maskChar
			continue
		}
		newStr = newStr + string(char)
	}

	return newStr
}

func (m *SecurityList) ColumSecurityApplyOnRecord(prevRecord reflect.Value, newRecord reflect.Value, modelType reflect.Type, pUserID int, pSchema, pTablename string) ([]string, error) {
	cols := make([]string, 0)
	if m.ColumnSecurity == nil {
		return cols, fmt.Errorf("security not initialized")
	}

	if prevRecord.Type() != newRecord.Type() {
		logger.Error("prev:%s and new:%s record type mismatch", prevRecord.Type(), newRecord.Type())
		return cols, fmt.Errorf("prev and new record type mismatch")
	}

	m.ColumnSecurityMutex.RLock()
	defer m.ColumnSecurityMutex.RUnlock()

	colsecList, ok := m.ColumnSecurity[fmt.Sprintf("%s.%s@%d", pSchema, pTablename, pUserID)]
	if !ok || colsecList == nil {
		return cols, fmt.Errorf("no security data")
	}

	for _, colsec := range colsecList {
		if !(strings.EqualFold(colsec.Accesstype, "mask") || strings.EqualFold(colsec.Accesstype, "hide")) {
			continue
		}
		lastRecords := interateStruct(prevRecord)
		newRecords := interateStruct(newRecord)
		var lastLoopField, lastLoopNewField reflect.Value
		pathLen := len(colsec.Path)
		for i, path := range colsec.Path {
			var nameType, fieldName string
			if len(newRecords) == 0 {
				if lastLoopNewField.IsValid() && lastLoopField.IsValid() && i < pathLen-1 {
					lastLoopNewField.Set(lastLoopField)
				}
				break
			}

			for ri := range newRecords {
				if !newRecords[ri].IsValid() || !lastRecords[ri].IsValid() {
					break
				}
				var field, oldField reflect.Value

				columnData := reflection.GetModelColumnDetail(newRecords[ri])
				lastColumnData := reflection.GetModelColumnDetail(lastRecords[ri])
				for i, cols := range columnData {
					if cols.SQLName != "" && strings.EqualFold(cols.SQLName, path) {
						nameType = "sql"
						fieldName = cols.SQLName
						field = cols.FieldValue
						oldField = lastColumnData[i].FieldValue
						break
					}
					if cols.Name != "" && strings.EqualFold(cols.Name, path) {
						nameType = "struct"
						fieldName = cols.Name
						field = cols.FieldValue
						oldField = lastColumnData[i].FieldValue
						break
					}
				}

				if !field.IsValid() || !oldField.IsValid() {
					break
				}
				lastLoopField = oldField
				lastLoopNewField = field

				if i == pathLen-1 {
					if strings.Contains(strings.ToLower(fieldName), "json") {
						prevSrc := oldField.Bytes()
						newSrc := field.Bytes()
						pathstr := strings.Join(colsec.Path, ".")
						prevPathValue := gjson.GetBytes(prevSrc, pathstr)
						newBytes, err := sjson.SetBytes(newSrc, pathstr, prevPathValue.Str)
						if err == nil {
							if field.CanSet() {
								field.SetBytes(newBytes)
							} else {
								logger.Warn("Value not settable: %v", field)
								cols = append(cols, pathstr)
							}
						}
						break
					}

					if nameType == "sql" {
						if strings.EqualFold(colsec.Accesstype, "mask") || strings.EqualFold(colsec.Accesstype, "hide") {
							field.Set(oldField)
							cols = append(cols, strings.Join(colsec.Path, "."))
						}
					}
					break
				}

				lastRecords = interateStruct(field)
				newRecords = interateStruct(oldField)
			}
		}
	}

	return cols, nil
}

func interateStruct(val reflect.Value) []reflect.Value {
	list := make([]reflect.Value, 0)

	switch val.Kind() {
	case reflect.Pointer, reflect.Interface:
		elem := val.Elem()
		if elem.IsValid() {
			list = append(list, interateStruct(elem)...)
		}
		return list
	case reflect.Array, reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if !elem.IsValid() {
				continue
			}
			list = append(list, interateStruct(elem)...)
		}
		return list
	case reflect.Struct:
		list = append(list, val)
		return list
	default:
		return list
	}
}

func setColSecValue(fieldsrc reflect.Value, colsec ColumnSecurity, fieldTypeName string) (int, reflect.Value) {
	fieldval := fieldsrc
	if fieldsrc.Kind() == reflect.Pointer || fieldsrc.Kind() == reflect.Interface {
		fieldval = fieldval.Elem()
	}

	if strings.Contains(strings.ToLower(fieldval.Kind().String()), "int") &&
		(strings.EqualFold(colsec.Accesstype, "mask") || strings.EqualFold(colsec.Accesstype, "hide")) {
		if fieldval.CanInt() && fieldval.CanSet() {
			fieldval.SetInt(0)
		}
	} else if (strings.Contains(strings.ToLower(fieldval.Kind().String()), "time") ||
		strings.Contains(strings.ToLower(fieldval.Kind().String()), "date")) &&
		(strings.EqualFold(colsec.Accesstype, "mask") || strings.EqualFold(colsec.Accesstype, "hide")) {
		fieldval.SetZero()
	} else if strings.Contains(strings.ToLower(fieldval.Kind().String()), "string") {
		strVal := fieldval.String()
		if strings.EqualFold(colsec.Accesstype, "mask") {
			fieldval.SetString(maskString(strVal, colsec.MaskStart, colsec.MaskEnd, colsec.MaskChar, colsec.MaskInvert))
		} else if strings.EqualFold(colsec.Accesstype, "hide") {
			fieldval.SetString("")
		}
	} else if strings.Contains(fieldTypeName, "json") &&
		(strings.EqualFold(colsec.Accesstype, "mask") || strings.EqualFold(colsec.Accesstype, "hide")) {
		if len(colsec.Path) < 2 {
			return 1, fieldval
		}
		pathstr := strings.Join(colsec.Path, ".")
		src := fieldval.Bytes()
		pathValue := gjson.GetBytes(src, pathstr)
		strValue := pathValue.String()
		if strings.EqualFold(colsec.Accesstype, "mask") {
			strValue = maskString(strValue, colsec.MaskStart, colsec.MaskEnd, colsec.MaskChar, colsec.MaskInvert)
		} else if strings.EqualFold(colsec.Accesstype, "hide") {
			strValue = ""
		}
		newBytes, err := sjson.SetBytes(src, pathstr, strValue)
		if err == nil {
			fieldval.SetBytes(newBytes)
		}
	}
	return 0, fieldsrc
}

func (m *SecurityList) ApplyColumnSecurity(records reflect.Value, modelType reflect.Type, pUserID int, pSchema, pTablename string) (error, reflect.Value) {
	defer logger.CatchPanic("ApplyColumnSecurity")

	if m.ColumnSecurity == nil {
		return fmt.Errorf("security not initialized"), records
	}

	m.ColumnSecurityMutex.RLock()
	defer m.ColumnSecurityMutex.RUnlock()

	colsecList, ok := m.ColumnSecurity[fmt.Sprintf("%s.%s@%d", pSchema, pTablename, pUserID)]
	if !ok || colsecList == nil {
		return fmt.Errorf("no security data"), records
	}

	for _, colsec := range colsecList {
		if !(strings.EqualFold(colsec.Accesstype, "mask") || strings.EqualFold(colsec.Accesstype, "hide")) {
			continue
		}

		if records.Kind() == reflect.Array || records.Kind() == reflect.Slice {
			for i := 0; i < records.Len(); i++ {
				record := records.Index(i)
				if !record.IsValid() {
					continue
				}

				lastRecord := interateStruct(record)
				pathLen := len(colsec.Path)
				for i, path := range colsec.Path {
					var field reflect.Value
					var nameType, fieldName string
					if len(lastRecord) == 0 {
						break
					}
					columnData := reflection.GetModelColumnDetail(lastRecord[0])
					for _, cols := range columnData {
						if cols.SQLName != "" && strings.EqualFold(cols.SQLName, path) {
							nameType = "sql"
							fieldName = cols.SQLName
							field = cols.FieldValue
							break
						}
						if cols.Name != "" && strings.EqualFold(cols.Name, path) {
							nameType = "struct"
							fieldName = cols.Name
							field = cols.FieldValue
							break
						}
					}

					if i == pathLen-1 {
						if nameType == "sql" || nameType == "struct" {
							setColSecValue(field, colsec, fieldName)
						}
						break
					}
					if field.IsValid() {
						lastRecord = interateStruct(field)
					}
				}
			}
		}
	}

	return nil, records
}

func (m *SecurityList) LoadColumnSecurity(pUserID int, pSchema, pTablename string, pOverwrite bool) error {
	// Use the callback if provided
	if m.LoadColumnSecurityCallback == nil {
		return fmt.Errorf("LoadColumnSecurityCallback not set - you must provide a callback function")
	}

	m.ColumnSecurityMutex.Lock()
	defer m.ColumnSecurityMutex.Unlock()

	if m.ColumnSecurity == nil {
		m.ColumnSecurity = make(map[string][]ColumnSecurity, 0)
	}
	secKey := fmt.Sprintf("%s.%s@%d", pSchema, pTablename, pUserID)

	if pOverwrite || m.ColumnSecurity[secKey] == nil {
		m.ColumnSecurity[secKey] = make([]ColumnSecurity, 0)
	}

	// Call the user-provided callback to load security rules
	colSecList, err := m.LoadColumnSecurityCallback(pUserID, pSchema, pTablename)
	if err != nil {
		return fmt.Errorf("LoadColumnSecurityCallback failed: %v", err)
	}

	m.ColumnSecurity[secKey] = colSecList
	return nil
}

func (m *SecurityList) ClearSecurity(pUserID int, pSchema, pTablename string) error {
	var filtered []ColumnSecurity
	m.ColumnSecurityMutex.Lock()
	defer m.ColumnSecurityMutex.Unlock()

	secKey := fmt.Sprintf("%s.%s@%d", pSchema, pTablename, pUserID)
	list, ok := m.ColumnSecurity[secKey]
	if !ok {
		return nil
	}

	for _, cs := range list {
		if !(cs.Schema == pSchema && cs.Tablename == pTablename && cs.UserID == pUserID) {
			filtered = append(filtered, cs)
		}
	}

	m.ColumnSecurity[secKey] = filtered
	return nil
}

func (m *SecurityList) LoadRowSecurity(pUserID int, pSchema, pTablename string, pOverwrite bool) (RowSecurity, error) {
	// Use the callback if provided
	if m.LoadRowSecurityCallback == nil {
		return RowSecurity{}, fmt.Errorf("LoadRowSecurityCallback not set - you must provide a callback function")
	}

	m.RowSecurityMutex.Lock()
	defer m.RowSecurityMutex.Unlock()

	if m.RowSecurity == nil {
		m.RowSecurity = make(map[string]RowSecurity, 0)
	}
	secKey := fmt.Sprintf("%s.%s@%d", pSchema, pTablename, pUserID)

	// Call the user-provided callback to load security rules
	record, err := m.LoadRowSecurityCallback(pUserID, pSchema, pTablename)
	if err != nil {
		return RowSecurity{}, fmt.Errorf("LoadRowSecurityCallback failed: %v", err)
	}

	m.RowSecurity[secKey] = record
	return record, nil
}

func (m *SecurityList) GetRowSecurityTemplate(pUserID int, pSchema, pTablename string) (RowSecurity, error) {
	defer logger.CatchPanic("GetRowSecurityTemplate")

	if m.RowSecurity == nil {
		return RowSecurity{}, fmt.Errorf("security not initialized")
	}

	m.RowSecurityMutex.RLock()
	defer m.RowSecurityMutex.RUnlock()

	rowSec, ok := m.RowSecurity[fmt.Sprintf("%s.%s@%d", pSchema, pTablename, pUserID)]
	if !ok {
		return RowSecurity{}, fmt.Errorf("no security data")
	}

	return rowSec, nil
}
