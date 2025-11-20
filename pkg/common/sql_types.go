package common

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func tryParseDT(str string) (time.Time, error) {
	var lasterror error
	tryFormats := []string{time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000",
		"06-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"02/01/2006",
		"02-01-2006",
		"2006-01-02",
		"15:04:05.000",
		"15:04:05",
		"15:04"}

	for _, f := range tryFormats {
		tx, err := time.Parse(f, str)
		if err == nil {
			return tx, nil
		} else {
			lasterror = err
		}
	}

	return time.Now(), lasterror
}

func ToJSONDT(dt time.Time) string {
	return dt.Format(time.RFC3339)
}

// SqlInt16 - A Int16 that supports SQL string
type SqlInt16 int16

// Scan -
func (n *SqlInt16) Scan(value interface{}) error {
	if value == nil {
		*n = 0
		return nil
	}
	switch v := value.(type) {
	case int:
		*n = SqlInt16(v)
	case int32:
		*n = SqlInt16(v)
	case int64:
		*n = SqlInt16(v)
	default:
		i, _ := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64)
		*n = SqlInt16(i)
	}

	return nil
}

// Value -
func (n SqlInt16) Value() (driver.Value, error) {
	if n == 0 {
		return nil, nil
	}
	return int64(n), nil
}

// String - Override String format of ZNullInt32
func (n SqlInt16) String() string {
	tmstr := fmt.Sprintf("%d", n)
	return tmstr
}

// UnmarshalJSON - Overre JidSON format of ZNullInt32
func (n *SqlInt16) UnmarshalJSON(b []byte) error {

	s := strings.Trim(strings.Trim(string(b), " "), "\"")

	n64, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*n = SqlInt16(n64)
	}

	return nil
}

// MarshalJSON - Override JSON format of time
func (n SqlInt16) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", n)), nil
}

// SqlInt32 - A int32 that supports SQL string
type SqlInt32 int32

// Scan -
func (n *SqlInt32) Scan(value interface{}) error {
	if value == nil {
		*n = 0
		return nil
	}
	switch v := value.(type) {
	case int:
		*n = SqlInt32(v)
	case int32:
		*n = SqlInt32(v)
	case int64:
		*n = SqlInt32(v)
	default:
		i, _ := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64)
		*n = SqlInt32(i)
	}

	return nil
}

// Value -
func (n SqlInt32) Value() (driver.Value, error) {
	if n == 0 {
		return nil, nil
	}
	return int64(n), nil
}

// String - Override String format of ZNullInt32
func (n SqlInt32) String() string {
	tmstr := fmt.Sprintf("%d", n)
	return tmstr
}

// UnmarshalJSON - Overre JidSON format of ZNullInt32
func (n *SqlInt32) UnmarshalJSON(b []byte) error {

	s := strings.Trim(strings.Trim(string(b), " "), "\"")

	n64, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*n = SqlInt32(n64)
	}

	return nil
}

// MarshalJSON - Override JSON format of time
func (n SqlInt32) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", n)), nil
}

// SqlInt64 - A int64 that supports SQL string
type SqlInt64 int64

// Scan -
func (n *SqlInt64) Scan(value interface{}) error {
	if value == nil {
		*n = 0
		return nil
	}
	switch v := value.(type) {
	case int:
		*n = SqlInt64(v)
	case int32:
		*n = SqlInt64(v)
	case uint32:
		*n = SqlInt64(v)
	case int64:
		*n = SqlInt64(v)
	case uint64:
		*n = SqlInt64(v)
	default:
		i, _ := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64)
		*n = SqlInt64(i)
	}

	return nil
}

// Value -
func (n SqlInt64) Value() (driver.Value, error) {
	if n == 0 {
		return nil, nil
	}
	return int64(n), nil
}

// String - Override String format of ZNullInt32
func (n SqlInt64) String() string {
	tmstr := fmt.Sprintf("%d", n)
	return tmstr
}

// UnmarshalJSON - Overre JidSON format of ZNullInt32
func (n *SqlInt64) UnmarshalJSON(b []byte) error {

	s := strings.Trim(strings.Trim(string(b), " "), "\"")

	n64, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*n = SqlInt64(n64)
	}

	return nil
}

// MarshalJSON - Override JSON format of time
func (n SqlInt64) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", n)), nil
}

// SqlTimeStamp - Implementation of SqlTimeStamp with some interfaces.
type SqlTimeStamp time.Time

// MarshalJSON - Override JSON format of time
func (t SqlTimeStamp) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	if time.Time(t).Before(time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return []byte("null"), nil
	}
	tmstr := time.Time(t).Format("2006-01-02T15:04:05")
	if tmstr == "0001-01-01T00:00:00" {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", tmstr)), nil
}

// UnmarshalJSON - Override JSON format of time
func (t *SqlTimeStamp) UnmarshalJSON(b []byte) error {
	var err error

	if b == nil {
		t = &SqlTimeStamp{}
		return nil
	}
	s := strings.Trim(strings.Trim(string(b), " "), "\"")
	if s == "null" || s == "" || s == "0" ||
		s == "0001-01-01T00:00:00" || s == "0001-01-01" {
		t = &SqlTimeStamp{}
		return nil
	}

	tx, err := tryParseDT(s)
	if err != nil {
		return err
	}

	*t = SqlTimeStamp(tx)
	return err
}

// Value - SQL Value of custom date
func (t SqlTimeStamp) Value() (driver.Value, error) {
	if t.GetTime().IsZero() || t.GetTime().Before(time.Date(0002, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return nil, nil
	}
	tmstr := time.Time(t).Format("2006-01-02T15:04:05")
	if tmstr <= "0001-01-01" || tmstr == "" {
		empty := time.Time{}
		return empty, nil
	}

	return tmstr, nil
}

// Scan - Scan custom date from sql
func (t *SqlTimeStamp) Scan(value interface{}) error {
	tm, ok := value.(time.Time)
	if ok {
		*t = SqlTimeStamp(tm)
		return nil
	}

	str, ok := value.(string)
	if ok {
		tx, err := tryParseDT(str)
		if err != nil {
			return err
		}
		*t = SqlTimeStamp(tx)
	}

	return nil
}

// String - Override String format of time
func (t SqlTimeStamp) String() string {
	return fmt.Sprintf("%s", time.Time(t).Format("2006-01-02T15:04:05"))
}

// GetTime - Returns Time
func (t SqlTimeStamp) GetTime() time.Time {
	return time.Time(t)
}

// SetTime - Returns Time
func (t *SqlTimeStamp) SetTime(pTime time.Time) {
	*t = SqlTimeStamp(pTime)
}

// Format - Formats the time
func (t SqlTimeStamp) Format(layout string) string {
	return fmt.Sprintf("%s", time.Time(t).Format(layout))
}

func SqlTimeStampNow() SqlTimeStamp {
	tx := time.Now()

	return SqlTimeStamp(tx)
}

// SqlFloat64 - SQL Int
type SqlFloat64 sql.NullFloat64

// Scan -
func (n *SqlFloat64) Scan(value interface{}) error {
	newval := sql.NullFloat64{Float64: 0, Valid: false}
	if value == nil {
		newval.Valid = false
		*n = SqlFloat64(newval)
		return nil
	}
	switch v := value.(type) {
	case int:
		newval.Float64 = float64(v)
		newval.Valid = true
	case float64:
		newval.Float64 = float64(v)
		newval.Valid = true
	case float32:
		newval.Float64 = float64(v)
		newval.Valid = true
	case int64:
		newval.Float64 = float64(v)
		newval.Valid = true
	case int32:
		newval.Float64 = float64(v)
		newval.Valid = true
	case uint16:
		newval.Float64 = float64(v)
		newval.Valid = true
	case uint64:
		newval.Float64 = float64(v)
		newval.Valid = true
	case uint32:
		newval.Float64 = float64(v)
		newval.Valid = true
	default:
		i, err := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64)
		newval.Float64 = float64(i)
		if err == nil {
			newval.Valid = false
		}
	}

	*n = SqlFloat64(newval)
	return nil
}

// Value -
func (n SqlFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return float64(n.Float64), nil
}

// String -
func (n SqlFloat64) String() string {
	if !n.Valid {
		return ""
	}
	tmstr := fmt.Sprintf("%f", n.Float64)
	return tmstr
}

// UnmarshalJSON -
func (n *SqlFloat64) UnmarshalJSON(b []byte) error {

	s := strings.Trim(strings.Trim(string(b), " "), "\"")
	invalid := (s == "null" || s == "" || len(s) < 2) || (strings.Contains(s, "{") || strings.Contains(s, "["))
	if invalid {
		return nil
	}

	nval, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	*n = SqlFloat64(sql.NullFloat64{Valid: true, Float64: float64(nval)})

	return nil
}

// MarshalJSON - Override JSON format of time
func (n SqlFloat64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("%f", n.Float64)), nil
}

// SqlDate - Implementation of SqlTime with some interfaces.
type SqlDate time.Time

// UnmarshalJSON - Override JSON format of time
func (t *SqlDate) UnmarshalJSON(b []byte) error {
	var err error

	s := strings.Trim(strings.Trim(string(b), " "), "\"")
	if s == "null" || s == "" || s == "0" ||
		strings.HasPrefix(s, "0001-01-01T00:00:00") ||
		s == "0001-01-01" {
		t = &SqlDate{}
		return nil
	}

	tx, err := tryParseDT(s)
	if err != nil {
		return err
	}
	*t = SqlDate(tx)
	return err
}

// MarshalJSON - Override JSON format of time
func (t SqlDate) MarshalJSON() ([]byte, error) {
	tmstr := time.Time(t).Format("2006-01-02") //time.RFC3339
	if strings.HasPrefix(tmstr, "0001-01-01") {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", tmstr)), nil
}

// Value - SQL Value of custom date
func (t SqlDate) Value() (driver.Value, error) {
	var s time.Time
	tmstr := time.Time(t).Format("2006-01-02")
	if strings.HasPrefix(tmstr, "0001-01-01") || tmstr <= "0001-01-01" {
		return nil, nil
	}
	s = time.Time(t)

	return s.Format("2006-01-02"), nil
}

// Scan - Scan custom date from sql
func (t *SqlDate) Scan(value interface{}) error {
	tm, ok := value.(time.Time)
	if ok {
		*t = SqlDate(tm)
		return nil
	}

	str, ok := value.(string)
	if ok {
		tx, err := tryParseDT(str)
		if err != nil {
			return err
		}

		*t = SqlDate(tx)
		return err
	}

	return nil
}

// Int64 - Override date format in unix epoch
func (t SqlDate) Int64() int64 {
	return time.Time(t).Unix()
}

// String - Override String format of time
func (t SqlDate) String() string {
	tmstr := time.Time(t).Format("2006-01-02") //time.RFC3339
	if strings.HasPrefix(tmstr, "0001-01-01") || strings.HasPrefix(tmstr, "1800-12-31") {
		return "0"
	}
	return tmstr
}

func SqlDateNow() SqlDate {
	tx := time.Now()
	return SqlDate(tx)
}

// ////////////////////// SqlTime /////////////////////////
// SqlTime - Implementation of SqlTime with some interfaces.
type SqlTime time.Time

// Int64 - Override Time format in unix epoch
func (t SqlTime) Int64() int64 {
	return time.Time(t).Unix()
}

// String - Override String format of time
func (t SqlTime) String() string {
	return time.Time(t).Format("15:04:05")
}

// UnmarshalJSON - Override JSON format of time
func (t *SqlTime) UnmarshalJSON(b []byte) error {
	var err error
	s := strings.Trim(strings.Trim(string(b), " "), "\"")
	if s == "null" || s == "" || s == "0" ||
		s == "0001-01-01T00:00:00" || s == "00:00:00" {
		*t = SqlTime{}
		return nil
	}
	tx := time.Time{}
	tx, err = tryParseDT(s)
	*t = SqlTime(tx)

	return err
}

// Format - Format Function
func (t SqlTime) Format(form string) string {
	tmstr := time.Time(t).Format(form)
	return tmstr
}

// Scan - Scan custom date from sql
func (t *SqlTime) Scan(value interface{}) error {
	tm, ok := value.(time.Time)
	if ok {
		*t = SqlTime(tm)
		return nil
	}

	str, ok := value.(string)
	if ok {
		tx, err := tryParseDT(str)
		*t = SqlTime(tx)
		return err
	}

	return nil
}

// Value - SQL Value of custom date
func (t SqlTime) Value() (driver.Value, error) {

	s := time.Time(t)
	st := s.Format("15:04:05")

	return st, nil
}

// MarshalJSON - Override JSON format of time
func (t SqlTime) MarshalJSON() ([]byte, error) {
	tmstr := time.Time(t).Format("15:04:05")
	if tmstr == "0001-01-01T00:00:00" || tmstr == "00:00:00" {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", tmstr)), nil
}

func SqlTimeNow() SqlTime {
	tx := time.Now()
	return SqlTime(tx)
}

// SqlJSONB - Nullable JSONB String
type SqlJSONB []byte

// Scan - Implements sql.Scanner for reading JSONB from database
func (n *SqlJSONB) Scan(value interface{}) error {
	if value == nil {
		*n = nil
		return nil
	}

	switch v := value.(type) {
	case string:
		*n = SqlJSONB([]byte(v))
	case []byte:
		*n = SqlJSONB(v)
	default:
		// For other types, marshal to JSON
		dat, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value to JSON: %v", err)
		}
		*n = SqlJSONB(dat)
	}

	return nil
}

// Value - Implements driver.Valuer for writing JSONB to database
func (n SqlJSONB) Value() (driver.Value, error) {
	if len(n) == 0 {
		return nil, nil
	}

	// Validate that it's valid JSON before returning
	var js interface{}
	if err := json.Unmarshal(n, &js); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}

	// Return as string for PostgreSQL JSONB/JSON columns
	return string(n), nil
}

func (n SqlJSONB) AsMap() (map[string]any, error) {
	if len(n) == 0 {
		return nil, nil
	}
	// Validate that it's valid JSON before returning
	js := make(map[string]any)
	if err := json.Unmarshal(n, &js); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	return js, nil
}

func (n SqlJSONB) AsSlice() ([]any, error) {
	if len(n) == 0 {
		return nil, nil
	}
	// Validate that it's valid JSON before returning
	js := make([]any, 0)
	if err := json.Unmarshal(n, &js); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	return js, nil
}

// UnmarshalJSON - Override JSON
func (n *SqlJSONB) UnmarshalJSON(b []byte) error {

	s := strings.Trim(strings.Trim(string(b), " "), "\"")
	invalid := (s == "null" || s == "" || len(s) < 2) || !(strings.Contains(s, "{") || strings.Contains(s, "["))
	if invalid {
		s = ""
		return nil
	}

	*n = []byte(s)

	return nil
}

// MarshalJSON - Override JSON format of time
func (n SqlJSONB) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte("null"), nil
	}
	var obj interface{}
	err := json.Unmarshal(n, &obj)
	if err != nil {
		//fmt.Printf("Invalid JSON %v", err)
		return []byte("null"), nil
	}

	// dat, err := json.MarshalIndent(obj, " ", " ")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to convert to JSON: %v", err)
	// }
	dat := n

	return dat, nil
}

// SqlUUID - Nullable UUID String
type SqlUUID sql.NullString

// Scan -
func (n *SqlUUID) Scan(value interface{}) error {
	str := sql.NullString{String: "", Valid: false}
	if value == nil {
		*n = SqlUUID(str)
		return nil
	}
	switch v := value.(type) {
	case string:
		uuid, err := uuid.Parse(v)
		if err == nil {
			str.String = uuid.String()
			str.Valid = true
			*n = SqlUUID(str)
		}
	case []uint8:
		uuid, err := uuid.ParseBytes(v)
		if err == nil {
			str.String = uuid.String()
			str.Valid = true
			*n = SqlUUID(str)
		}
	default:
		uuid, err := uuid.Parse(fmt.Sprintf("%v", v))
		if err == nil {
			str.String = uuid.String()
			str.Valid = true
			*n = SqlUUID(str)
		}
	}

	return nil
}

// Value -
func (n SqlUUID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.String, nil
}

// UnmarshalJSON - Override JSON
func (n *SqlUUID) UnmarshalJSON(b []byte) error {

	s := strings.Trim(strings.Trim(string(b), " "), "\"")
	invalid := (s == "null" || s == "" || len(s) < 30)
	if invalid {
		s = ""
		return nil
	}
	*n = SqlUUID(sql.NullString{String: s, Valid: !invalid})

	return nil
}

// MarshalJSON - Override JSON format of time
func (n SqlUUID) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", n.String)), nil
}

// TryIfInt64 - Wrapper function to quickly try and cast text to int
func TryIfInt64(v any, def int64) int64 {
	str := ""
	switch val := v.(type) {
	case string:
		str = val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case []byte:
		str = string(val)
	default:
		str = fmt.Sprintf("%d", def)
	}
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return def
	}
	return val
}
