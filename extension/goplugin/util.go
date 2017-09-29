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
	"go/types"
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
	if anyOfKinds(resource.Kind(), primitiveTypes) {
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
			return fmt.Errorf("missing tag 'json' for resource %s field %s", resource.Type().Name(), fieldType.Name)
		}
		kind := fieldType.Type.Kind()
		if kind == reflect.Interface {
			if field.IsNil() && context[propertyName] == nil {
				continue
			}
			field.Set(reflect.ValueOf(context[propertyName]))
		} else if anyOfKinds(kind, []reflect.Kind{reflect.Ptr, reflect.Struct}) && strings.HasPrefix(strings.TrimLeft(fieldType.Type.String(), "*"), "goext.Null") {
			if kind == reflect.Ptr {
				if _, ok := context[propertyName]; !ok {
					continue
				}
				field.Set(reflect.New(fieldType.Type.Elem()))
				field = field.Elem()
			}
			if context[propertyName] == nil {
				field.FieldByName("Valid").SetBool(false)
			} else {
				field.FieldByName("Valid").SetBool(true)
				valueField := field.FieldByName("Value")
				value := reflect.ValueOf(context[propertyName])
				if value.Type() == valueField.Type() {
					valueField.Set(value)
				} else {
					if valueField.Kind() == reflect.Int && value.Kind() == reflect.Float64 { // reflect treats number(N, 0) as float
						valueField.SetInt(int64(value.Float()))
					} else {
						return fmt.Errorf("invalid type of '%s' field (%s, expecting %s)", propertyName, value.Kind(), valueField.Kind())
					}
				}
			}
		} else if kind == reflect.Struct || kind == reflect.Ptr {
			if context[propertyName] != nil {
				field.Set(reflect.ValueOf(reflect.New(field.Type()).Elem().Interface()))
				if err := resourceFromMap(context[propertyName].(map[string]interface{}), field); err != nil {
					return err
				}
			}
		} else if kind == reflect.Slice {
			if v, ok := context[propertyName]; ok {
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
				case types.Nil:
					field.Set(reflect.Zero(field.Type()))
					continue
				default:
					val := reflect.ValueOf(v)
					if !val.IsValid() {
						field.Set(reflect.Zero(field.Type()))
						continue
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
		} else {
			value := reflect.ValueOf(context[propertyName])
			if value.IsValid() {
				if value.Type() == field.Type() {
					field.Set(value)
				} else {
					if field.Kind() == reflect.Int && value.Kind() == reflect.Float64 { // reflect treats number(N, 0) as float
						field.SetInt(int64(value.Float()))
					} else {
						return fmt.Errorf("invalid type of '%s' field (%s, expecting %s)", propertyName, value.Kind(), field.Kind())
					}
				}
			}
		}
	}

	return nil
}

func anyOfKinds(kind reflect.Kind, kinds []reflect.Kind) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

var (
	nullableKinds = []reflect.Kind{
		reflect.Map,
		reflect.Slice,
		reflect.Interface,
		reflect.Ptr,
	}

	primitiveTypes = []reflect.Kind{
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.String,
	}
)

// ResourceToMap converts structure representation of the resource to mapped representation
func (util *Util) ResourceToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("json")
	structMap := mapper.TypeMap(reflect.TypeOf(resource))
	resourceValue := reflect.ValueOf(resource).Elem()

	for field, fi := range structMap.Names {
		if len(fi.Index) != 1 {
			continue
		}

		v := resourceValue.FieldByIndex(fi.Index)
		val := v.Interface()
		if field == "id" && v.String() == "" {
			id := uuid.NewV4().String()
			fieldsMap[field] = id
			v.SetString(id)
		} else if anyOfKinds(v.Kind(), []reflect.Kind{reflect.Ptr, reflect.Struct}) && strings.HasPrefix(strings.TrimLeft(v.Type().String(), "*"), "goext.Null") {
			if v.Kind() == reflect.Ptr {
				if !v.IsNil() {
					v = v.Elem()
				} else {
					// nothing, undefined value shoudn't appear in map
					continue
				}
			}

			valid := v.FieldByName("Valid").Bool()
			if valid {
				fieldsMap[field] = v.FieldByName("Value").Interface()
			} else {
				fieldsMap[field] = nil
			}
		} else if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				fieldsMap[field] = nil
			} else {
				rv := util.ResourceToMap(val)
				fieldsMap[field] = rv
			}
		} else if v.Kind() == reflect.Slice {
			if v.IsNil() {
				// nothing
			} else {
				sliceElem := v.Type().Elem()
				if anyOfKinds(sliceElem.Kind(), primitiveTypes) {
					slice := make([]interface{}, v.Len())
					for i := 0; i < v.Len(); i++ {
						slice[i] = v.Index(i).Interface()
					}
					fieldsMap[field] = slice
				} else {
					slice := make([]map[string]interface{}, v.Len())
					for i := 0; i < v.Len(); i++ {
						elem := v.Index(i)
						if anyOfKinds(elem.Kind(), nullableKinds) {
							if !elem.IsNil() {
								slice[i] = util.ResourceToMap(elem.Interface())
							} else {
								slice[i] = nil
							}
						} else if elem.Kind() == reflect.Struct {
							slice[i] = util.ResourceToMap(elem.Addr().Interface())
						}
					}
					fieldsMap[field] = slice
				}
			}
		} else if v.Kind() == reflect.Struct {
			fieldsMap[field] = util.ResourceToMap(v.Addr().Interface())
		} else {
			fieldsMap[field] = val
		}
	}

	return fieldsMap
}
