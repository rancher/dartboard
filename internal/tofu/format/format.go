/*
Copyright © 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


Copyright 2018 Gruntwork, Inc.

This product includes modified software developed at Gruntwork (https://www.gruntwork.io/).
*/

package format

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// Try to convert the given value to a generic slice. Return the slice and true if the underlying value itself was a
// slice and an empty slice and false if it wasn't.
func toSliceOfAny(value any) ([]interface{}, bool) {
	reflectValue := reflect.ValueOf(value)
	if reflectValue.Kind() != reflect.Slice {
		return []interface{}{}, false
	}

	genericSlice := make([]interface{}, reflectValue.Len())

	for i := 0; i < reflectValue.Len(); i++ {
		genericSlice[i] = reflectValue.Index(i).Interface()
	}

	return genericSlice, true
}

// Try to convert the given value to a generic map. Return the map and true if the underlying value itself was a
// map and an empty map and false if it wasn't
func toMapOfAny(value any) (map[string]interface{}, bool) {
	reflectValue := reflect.ValueOf(value)
	if reflectValue.Kind() != reflect.Map {
		return map[string]interface{}{}, false
	}

	reflectType := reflect.TypeOf(value)
	if reflectType.Key().Kind() != reflect.String {
		return map[string]interface{}{}, false
	}

	genericMap := make(map[string]interface{}, reflectValue.Len())

	mapKeys := reflectValue.MapKeys()
	for _, key := range mapKeys {
		genericMap[key.String()] = reflectValue.MapIndex(key).Interface()
	}

	return genericMap, true
}

// Convert a slice to an HCL string. See ToHclString for details.
func sliceToHclString(slice []any) string {
	hclValues := []string{}

	for _, value := range slice {
		hclValue := convertValueToHCL(value, true)
		hclValues = append(hclValues, hclValue)
	}

	return fmt.Sprintf("[%s]", strings.Join(hclValues, ", "))
}

// Convert a map to an HCL string. See ToHclString for details.
func mapToHclString(m map[string]any) string {
	keyValuePairs := []string{}

	for key, value := range m {
		keyValuePair := fmt.Sprintf(`"%s" = %s`, key, convertValueToHCL(value, true))
		keyValuePairs = append(keyValuePairs, keyValuePair)
	}

	return fmt.Sprintf("{%s}", strings.Join(keyValuePairs, ", "))
}

// Convert a primitive, such as a bool, int, or string, to an HCL string. If this isn't a primitive, force its value
// using Sprintf. See ToHclString for details.
func primitiveToHclString(value interface{}, isNested bool) (string, error) {
	if value == nil {
		return "", errors.New("unable to parse value of type nil")
	}

	switch v := value.(type) {
	case string:
		if isNested {
			return fmt.Sprintf("\"%v\"", v), nil
		}
		return fmt.Sprintf("%v", v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case int, int32, int64:
		// explicitly convert to int64 if needed
		vInt64, ok := v.(int64)
		if !ok {
			vInt32, ok := v.(int32)
			if !ok {
				vInt64 = int64(v.(int))
			} else {
				vInt64 = int64(vInt32)
			}
		}
		return fmt.Sprintf("%d", vInt64), nil
	case float32, float64:
		// explicitly convert to float64 if needed
		vFloat64, ok := v.(float64)
		if !ok {
			vFloat64 = float64(v.(float32))
			return strconv.FormatFloat(vFloat64, 'f', -1, 32), nil
		}
		return strconv.FormatFloat(vFloat64, 'f', -1, 64), nil
	default:
		return fmt.Sprintf("%v", v), fmt.Errorf("no defined case for type of value: %T", v)
	}
}

// converts golang types into HCL format utilizing the hclwrite and cty packages
// reference: https://pkg.go.dev/github.com/hashicorp/hcl/v2/hclwrite#example-package-GenerateFromScratch
func convertValueToHCL(value any, isNested bool) string {
	var v string
	var err error
	if slice, isSlice := toSliceOfAny(value); isSlice {
		v = sliceToHclString(slice)
	} else if m, isMap := toMapOfAny(value); isMap {
		v = mapToHclString(m)
	} else {
		v, err = primitiveToHclString(value, isNested)
	}
	if err != nil {
		log.Panicf("%v", err)
	}
	return v
}
