package reflection

import (
	"reflect"
	"strings"

	"github.com/bitechdev/ResolveSpec/pkg/logger"
)

type ModelFieldDetail struct {
	Name        string        `json:"name"`
	DataType    string        `json:"datatype"`
	SQLName     string        `json:"sqlname"`
	SQLDataType string        `json:"sqldatatype"`
	SQLKey      string        `json:"sqlkey"`
	Nullable    bool          `json:"nullable"`
	FieldValue  reflect.Value `json:"-"`
}

// GetModelColumnDetail - Get a list of columns in the SQL declaration of the model
func GetModelColumnDetail(record reflect.Value) []ModelFieldDetail {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in GetModelColumnDetail : %v", r)
		}
	}()

	var lst []ModelFieldDetail
	lst = make([]ModelFieldDetail, 0)

	if !record.IsValid() {
		return lst
	}
	if record.Kind() == reflect.Pointer || record.Kind() == reflect.Interface {
		record = record.Elem()
	}
	if record.Kind() != reflect.Struct {
		return lst
	}
	modeltype := record.Type()

	for i := 0; i < modeltype.NumField(); i++ {
		fieldtype := modeltype.Field(i)
		gormdetail := fieldtype.Tag.Get("gorm")
		gormdetail = strings.Trim(gormdetail, " ")
		fielddetail := ModelFieldDetail{}
		fielddetail.FieldValue = record.Field(i)
		fielddetail.Name = fieldtype.Name
		fielddetail.DataType = fieldtype.Type.Name()
		fielddetail.SQLName = fnFindKeyVal(gormdetail, "column:")
		fielddetail.SQLDataType = fnFindKeyVal(gormdetail, "type:")
		if strings.Index(strings.ToLower(gormdetail), "identity") > 0 ||
			strings.Index(strings.ToLower(gormdetail), "primary_key") > 0 {
			fielddetail.SQLKey = "primary_key"
		} else if strings.Contains(strings.ToLower(gormdetail), "unique") {
			fielddetail.SQLKey = "unique"
		} else if strings.Contains(strings.ToLower(gormdetail), "uniqueindex") {
			fielddetail.SQLKey = "uniqueindex"
		}

		if strings.Contains(strings.ToLower(gormdetail), "nullable") {
			fielddetail.Nullable = true
		} else if strings.Contains(strings.ToLower(gormdetail), "null") {
			fielddetail.Nullable = true
		}
		if strings.Contains(strings.ToLower(gormdetail), "not null") {
			fielddetail.Nullable = false
		}

		if strings.Contains(strings.ToLower(gormdetail), "foreignkey:") {
			fielddetail.SQLKey = "foreign_key"
			ik := strings.Index(strings.ToLower(gormdetail), "foreignkey:")
			ie := strings.Index(gormdetail[ik:], ";")
			if ie > ik && ik > 0 {
				fielddetail.SQLName = strings.ToLower(gormdetail)[ik+11 : ik+ie]
				// fmt.Printf("\r\nforeignkey: %v", fielddetail)
			}

		}
		// ";foreignkey:rid_parent;association_foreignkey:id_atevent;save_associations:false;association_autocreate:false;"

		lst = append(lst, fielddetail)

	}
	return lst
}

func fnFindKeyVal(src, key string) string {
	icolStart := strings.Index(strings.ToLower(src), strings.ToLower(key))
	val := ""
	if icolStart >= 0 {
		val = src[icolStart+len(key):]
		icolend := strings.Index(val, ";")
		if icolend > 0 {
			val = val[:icolend]
		}
		return val
	}
	return ""
}
