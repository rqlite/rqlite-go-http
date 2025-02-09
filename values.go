package http

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

// MakeURLValues converts a struct to a url.Values, using the `uvalue` tag to
// determine the key name.
func MakeURLValues(input any) (url.Values, error) {
	vals := url.Values{}
	if input == nil {
		return vals, nil
	}

	val := reflect.ValueOf(input)
	typ := reflect.TypeOf(input)

	// If it's a pointer, get the underlying element.
	if typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return vals, nil
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a pointer to a struct, got %s", typ.Kind())
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagVal := field.Tag.Get("uvalue")
		if tagVal == "" {
			// No `uvalue` tag, skip.
			continue
		}

		fieldValue := val.Field(i)
		if !fieldValue.CanInterface() {
			// Unexported or inaccessible field.
			continue
		}

		var strVal string
		if fieldValue.Type() == reflect.TypeOf(time.Duration(0)) {
			d := fieldValue.Interface().(time.Duration)
			strVal = d.String()
		} else {
			switch fieldValue.Kind() {
			case reflect.String:
				strVal = fieldValue.Interface().(string)
			case reflect.Bool:
				b := fieldValue.Interface().(bool)
				strVal = strconv.FormatBool(b)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				i := fieldValue.Int()
				strVal = strconv.FormatInt(i, 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				u := fieldValue.Uint()
				strVal = strconv.FormatUint(u, 10)
			default:
				continue
			}
		}
		vals.Add(tagVal, strVal)
	}
	return vals, nil
}
