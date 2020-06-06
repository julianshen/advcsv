package advcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
)

type UnsupportedTypeError struct {
	Type reflect.Type
}

type csvField struct {
	headTitle  string
	index      int
	fieldIndex []int
}

type CustomCSVType interface {
	UnmarshalCSV(data string) error
}

var customCsvType = reflect.TypeOf(new(CustomCSVType)).Elem()

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("advcsv: unsupported type: %v", e.Type)
}

func validateType(v interface{}) error {
	val := reflect.ValueOf(v)

	if v == nil || val.Kind() != reflect.Ptr || val.IsNil() || val.Type().Elem().Kind() != reflect.Slice {
		return &UnsupportedTypeError{
			reflect.TypeOf(v),
		}
	}

	arrayElem := val.Type().Elem().Elem()

	if arrayElem.Kind() != reflect.Struct && !(arrayElem.Kind() == reflect.Ptr && arrayElem.Elem().Kind() == reflect.Struct) {
		return &UnsupportedTypeError{
			reflect.TypeOf(v),
		}
	}

	return nil
}

func constructCsvFields(headers []string, t reflect.Type) []*csvField {
	var structType reflect.Type

	if t.Kind() == reflect.Ptr {
		structType = t.Elem()
	} else {
		structType = t
	}

	csvFields := make([]*csvField, 0)
	numFields := structType.NumField()

	idxmap := make(map[string]int)
	for i, h := range headers {
		idxmap[h] = i
	}

	for i := 0; i < numFields; i++ {
		if tag, ok := structType.Field(i).Tag.Lookup("csv"); ok {
			if idx, ok := idxmap[tag]; ok {
				csvFields = append(csvFields, &csvField{
					headTitle:  tag,
					index:      idx,
					fieldIndex: structType.Field(i).Index,
				})
			}
		}
	}

	return csvFields
}

func unmarshalRecord(record []string, elemType reflect.Type, csvFields []*csvField) (reflect.Value, error) {
	var t reflect.Type
	if elemType.Kind() == reflect.Ptr {
		t = elemType.Elem()
	} else {
		t = elemType
	}

	val := reflect.New(t)

	for _, csvField := range csvFields {
		field := val.Elem().FieldByIndex(csvField.fieldIndex)

		if field.Type().Implements(customCsvType) && field.Kind() == reflect.Ptr {
			fieldVal := reflect.New(field.Type().Elem())
			field.Set(fieldVal)
			fieldVal.MethodByName("UnmarshalCSV").Call([]reflect.Value{reflect.ValueOf(record[csvField.index])})
		} else if field.Kind() == reflect.String {
			field.Set(reflect.ValueOf(record[csvField.index]))
		} else {
			return reflect.ValueOf(nil), &UnsupportedTypeError{}
		}
	}

	if elemType.Kind() == reflect.Ptr {
		return val, nil
	}
	return val.Elem(), nil
}

func Unmarshal(r io.Reader, v interface{}) error {
	if err := validateType(v); err != nil {
		return err
	}

	val := reflect.ValueOf(v)

	csvReader := csv.NewReader(r)
	csvReader.LazyQuotes = true
	csvReader.Comment = '#'

	headers, err := csvReader.Read()

	if err != nil {
		return err
	}

	elemType := reflect.TypeOf(v).Elem().Elem()

	csvFields := constructCsvFields(headers, elemType)

	for {
		if record, err := csvReader.Read(); err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		} else {
			data, err := unmarshalRecord(record, elemType, csvFields)
			if err != nil {
				return err
			}
			val.Elem().Set(reflect.Append(val.Elem(), data))
		}
	}

	return nil
}
