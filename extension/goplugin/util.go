// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goplugin

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/twinj/uuid"
)

type Util struct {
}

func contextGetTransaction(ctx goext.Context) (goext.ITransaction, bool) {
	ctxTx := ctx["transaction"]
	if ctxTx == nil {
		return nil, false
	}

	switch tx := ctxTx.(type) {
	case goext.ITransaction:
		return tx, true
	case transaction.Transaction:
		return &Transaction{tx}, true
	default:
		panic(fmt.Sprintf("Unknown transaction type in context: %+v", ctxTx))
	}
}

// NewUUID create a new unique ID
func (util *Util) NewUUID() string {
	return uuid.NewV4().String()
}

func (u *Util) GetTransaction(context goext.Context) (goext.ITransaction, bool) {
	return contextGetTransaction(context)
}

func (u *Util) Clone() *Util {
	return &Util{}
}

var controllers map[gomock.TestReporter]*gomock.Controller = make(map[gomock.TestReporter]*gomock.Controller)

func NewController(testReporter gomock.TestReporter) *gomock.Controller {
	ctrl := gomock.NewController(testReporter)
	controllers[testReporter] = ctrl
	return ctrl
}

func Finish(testReporter gomock.TestReporter) {
	controllers[testReporter].Finish()
}

// ResourceFromMapForType converts mapped representation to structure representation of the resource for given type
func (util *Util) ResourceFromMapForType(context map[string]interface{}, rawResource interface{}) (goext.Resource, error) {
	resource := reflect.New(reflect.TypeOf(rawResource))
	if err := resourceFromMap(context, resource); err != nil {
		return nil, err
	}
	return resource.Interface(), nil
}

// expects primitive or allocated pointer to struct
func resourceFromMap(context map[string]interface{}, resource reflect.Value) error {
	if isPrimitiveKind(resource.Kind()) {
		resource.Set(reflect.ValueOf(context))
		return nil
	}
	if resource.Kind() == reflect.Ptr {
		if context == nil && resource.IsNil() {
			return nil
		} else if context != nil && resource.IsNil() {
			resource.Set(reflect.New(resource.Type().Elem()))
			return resourceFromMap(context, resource.Elem())
		}
		return resourceFromMap(context, resource.Elem())

	}
	for i := 0; i < resource.NumField(); i++ {
		field := resource.Field(i)
		fieldType := resource.Type().Field(i)
		propertyName := strings.Split(fieldType.Tag.Get("json"), ",")[0]
		if propertyName == "" {
			return fmt.Errorf("missing tag 'json' for resource %s field %s", resource.Type().String(), fieldType.Name)
		}
		kind := fieldType.Type.Kind()
		mapValue, mapValueExists := context[propertyName]

		if kind == reflect.Interface {
			if field.IsNil() && mapValue == nil {
				continue
			}
			field.Set(reflect.ValueOf(mapValue))
		} else if isMaybeType(kind, fieldType.Type.String()) {
			if kind == reflect.Ptr {
				if !mapValueExists {
					continue
				}
				field.Set(reflect.New(fieldType.Type.Elem()))
				field = field.Elem()
			}
			if err := maybeFromMap(context, propertyName, field); err != nil {
				return err
			}
		} else if kind == reflect.Struct || kind == reflect.Ptr {
			if mapValue != nil {
				field.Set(reflect.ValueOf(reflect.New(field.Type()).Elem().Interface()))
				if err := resourceFromMap(mapValue.(map[string]interface{}), field); err != nil {
					return err
				}
			}
		} else if kind == reflect.Slice {
			if err := sliceToMap(context, propertyName, field); err != nil {
				return err
			}
		} else {
			if err := assignMapValueToField(mapValue, propertyName, field); err != nil {
				return err
			}
		}
	}

	return nil
}

func assignMapValueToField(mapValue interface{}, fieldName string, field reflect.Value) error {
	value := reflect.ValueOf(mapValue)
	if value.IsValid() {
		if value.Type() == field.Type() {
			field.Set(value)
		} else {
			if field.Kind() == reflect.Int && value.Kind() == reflect.Float64 { // reflect treats number(N, 0) as float
				field.SetInt(int64(value.Float()))
			} else {
				return fmt.Errorf("invalid type of '%s' field (%s, expecting %s)", fieldName, value.Kind(), field.Kind())
			}
		}
	}
	return nil
}

func isMaybeType(kind reflect.Kind, typeName string) bool {
	return anyOfKinds(kind, []reflect.Kind{reflect.Ptr, reflect.Struct}) && strings.HasPrefix(typeName, "goext.Maybe")
}

func maybeFromMap(context map[string]interface{}, fieldName string, field reflect.Value) error {
	if mapValue, ok := context[fieldName]; !ok {
		field.FieldByName("State").SetInt(int64(goext.MaybeUndefined))
	} else if mapValue == nil {
		field.FieldByName("State").SetInt(int64(goext.MaybeNull))
	} else {
		field.FieldByName("State").SetInt(int64(goext.MaybeNotNull))
		if err := assignMapValueToField(mapValue, fieldName, field.FieldByName("Value")); err != nil {
			return err
		}
	}
	return nil
}

func sliceToMap(context map[string]interface{}, fieldName string, field reflect.Value) error {
	if v, ok := context[fieldName]; ok {
		sliceElems := 0
		interfaces := false
		structures := false
		switch v.(type) {
		case []map[string]interface{}:
			sliceElems = len(v.([]map[string]interface{}))
			structures = true
		case []interface{}:
			sliceElems = len(v.([]interface{}))
			interfaces = true
		default:
			val := reflect.ValueOf(v)
			if !val.IsValid() {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}
			sliceElems = val.Len()
		}
		field.Set(reflect.MakeSlice(field.Type(), sliceElems, sliceElems))
		field.SetLen(sliceElems)
		for i := 0; i < sliceElems; i++ {
			elemType := field.Type().Elem()
			elem := field.Index(i)
			nestedField := reflect.New(elemType).Elem()
			if structures {
				if err := resourceFromMap(v.([]map[string]interface{})[i], nestedField); err != nil {
					return err
				}
			} else if interfaces {
				nestedValue := v.([]interface{})[i]
				if nestedMap, ok := nestedValue.(map[string]interface{}); ok {
					if err := resourceFromMap(nestedMap, nestedField); err != nil {
						return err
					}
				} else {
					nestedField.Set(reflect.ValueOf(nestedValue))
				}
			} else {
				val := reflect.ValueOf(v)
				nestedField.Set(val.Index(i))
			}
			elem.Set(nestedField)
		}
	}
	return nil
}

// ResourceToMap converts structure representation of the resource to mapped representation
func (util *Util) ResourceToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("json")
	structMap := mapper.TypeMap(reflect.TypeOf(resource))
	resourceValue := reflect.ValueOf(resource).Elem()

	for fieldName, fi := range structMap.Names {
		if len(fi.Index) != 1 {
			continue
		}

		v := resourceValue.FieldByIndex(fi.Index)
		val := v.Interface()
		if fieldName == "id" && v.String() == "" {
			id := uuid.NewV4().String()
			fieldsMap[fieldName] = id
			v.SetString(id)
		} else if isMaybeType(v.Kind(), v.Type().String()) {
			maybeToMap(fieldsMap, fieldName, v)
		} else if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				fieldsMap[fieldName] = nil
			} else {
				rv := util.ResourceToMap(val)
				fieldsMap[fieldName] = rv
			}
		} else if v.Kind() == reflect.Slice {
			util.sliceToMap(fieldsMap, fieldName, v)
		} else if v.Kind() == reflect.Struct {
			fieldsMap[fieldName] = util.ResourceToMap(v.Addr().Interface())
		} else {
			fieldsMap[fieldName] = val
		}
	}

	return fieldsMap
}

func (util *Util) sliceToMap(fieldsMap map[string]interface{}, fieldName string, v reflect.Value) {
	if v.IsNil() {
		// nothing
	} else {
		sliceElem := v.Type().Elem()
		if isPrimitiveKind(sliceElem.Kind()) {
			slice := make([]interface{}, v.Len())
			for i := 0; i < v.Len(); i++ {
				slice[i] = v.Index(i).Interface()
			}
			fieldsMap[fieldName] = slice
		} else {
			slice := make([]map[string]interface{}, v.Len())
			for i := 0; i < v.Len(); i++ {
				elem := v.Index(i)
				if isNullableKind(elem.Kind()) {
					if !elem.IsNil() {
						slice[i] = util.ResourceToMap(elem.Interface())
					} else {
						slice[i] = nil
					}
				} else if elem.Kind() == reflect.Struct {
					slice[i] = util.ResourceToMap(elem.Addr().Interface())
				}
			}
			fieldsMap[fieldName] = slice
		}
	}
}

func maybeToMap(fieldsMap map[string]interface{}, fieldName string, value reflect.Value) {
	if value.Kind() == reflect.Ptr {
		if !value.IsNil() {
			value = value.Elem()
		} else {
			// nothing, undefined value shoudn't appear in map
			return
		}
	}

	state := value.FieldByName("State").Int()
	switch goext.MaybeState(state) {
	case goext.MaybeUndefined:
		// nothing
	case goext.MaybeNull:
		fieldsMap[fieldName] = nil
	case goext.MaybeNotNull:
		fieldsMap[fieldName] = value.FieldByName("Value").Interface()
	}
}

func anyOfKinds(kind reflect.Kind, kinds []reflect.Kind) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func isNullableKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Map:
		fallthrough
	case reflect.Slice:
		fallthrough
	case reflect.Interface:
		fallthrough
	case reflect.Ptr:
		return true
	}
	return false
}

func isPrimitiveKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Uintptr:
		fallthrough
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		fallthrough
	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		fallthrough
	case reflect.String:
		return true
	}
	return false
}
