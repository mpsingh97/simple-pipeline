package models

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

func dereferenceValue(val interface{}) interface{} {
	if valPtr, ok := val.(*interface{}); ok {
		return *valPtr
	}
	return val
}

func Decode(data []interface{}, result interface{}) error {
	v := reflect.ValueOf(result)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("result argument must be a pointer to a struct")
	}

	v = v.Elem()

	if len(data) < v.NumField() {
		return fmt.Errorf("data length (%d) is less than the struct field count (%d)", len(data), v.NumField())
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %d of struct", i)
		}

		rawValue := dereferenceValue(data[i])
		value := reflect.ValueOf(rawValue)

		if !value.Type().AssignableTo(field.Type()) {
			if field.Kind() == reflect.String {
				if str, ok := rawValue.(string); ok {
					field.SetString(str)
				} else {
					return fmt.Errorf("cannot assign value of type %T to field %s", rawValue, field.Type())
				}
			} else if field.Kind() == reflect.Int {
				if val, ok := rawValue.(int); ok {
					field.SetInt(int64(val))
				} else if val, ok := rawValue.(int64); ok {
					field.SetInt(val)
				} else if val, ok := rawValue.(int32); ok {
					field.SetInt(int64(val))
				} else {
					return fmt.Errorf("cannot assign value of type %T to field %s", rawValue, field.Type())
				}
			} else if field.Kind() == reflect.Struct && field.Type() == reflect.TypeOf(time.Time{}) {
				if t, ok := rawValue.(time.Time); ok {
					field.Set(reflect.ValueOf(t))
				} else {
					return fmt.Errorf("cannot assign value of type %T to field %s", rawValue, field.Type())
				}
			} else {
				return fmt.Errorf("cannot assign value of type %T to field %s", rawValue, field.Type())
			}
		} else {
			field.Set(value)
		}
	}

	return nil
}
