package objfilter

import (
	"fmt"
	"reflect"
	"strings"
)

type jsonObject struct {
	obj    any
	fields map[string]any
}

func NewJsonObject(obj any) Object {
	return &jsonObject{obj: obj, fields: nil}
}

func (obj *jsonObject) GetValue(name string) Value {
	if obj.fields == nil {
		obj.fields = make(map[string]any)
		err := jsonConvertToMap(obj.obj, obj.fields)
		if err != nil {
			// objects should always be marshalable
			panic(err)
		}
	}

	val := Value{}
	val.V, val.Ok = obj.fields[name]
	return val
}

func jsonConvertToMap(from any, to map[string]any) error {
	val := reflect.ValueOf(from)
	typ := reflect.TypeOf(from)

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", val.Kind().String())
	}

	for i := 0; i < typ.NumField(); i++ {
		jsonFieldName := getJsonFieldName(typ.Field(i))
		switch typ.Field(i).Type.Kind() {
		case reflect.Bool:
			to[jsonFieldName] = val.Field(i).Bool()
		case reflect.Int:
			fallthrough
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			to[jsonFieldName] = val.Field(i).Int()
		case reflect.Uint:
			fallthrough
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			fallthrough
		case reflect.Uint64:
			to[jsonFieldName] = val.Field(i).Uint()
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			to[jsonFieldName] = val.Field(i).Float()
		case reflect.Array:
			to[jsonFieldName] = val.Field(i).Interface()
		case reflect.Map:
			to[jsonFieldName] = val.Field(i).Interface()
		case reflect.Pointer:
			to[jsonFieldName] = val.Field(i).Interface()
		case reflect.Slice:
			to[jsonFieldName] = val.Field(i).Interface()
		case reflect.String:
			to[jsonFieldName] = val.Field(i).String()
		case reflect.Struct:
			to[jsonFieldName] = val.Field(i).Interface()
		default:
			return fmt.Errorf("unsupported field type %s", typ.Field(i).Type.Kind().String())
		}
	}

	return nil
}

func getJsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}

	fieldName, _, _ := strings.Cut(tag, ",")
	return fieldName
}
