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
	"strconv"
	"strings"
)

// Convert a slice to an HCL string. See ConvertValueToHCL for details.
func sliceToHclString(slice []any) string {
	hclValues := []string{}

	for _, value := range slice {
		hclValue := ConvertValueToHCL(value, true)
		hclValues = append(hclValues, hclValue)
	}

	return fmt.Sprintf("[%s]", strings.Join(hclValues, ", "))
}

// Convert a map to an HCL string. See ConvertValueToHCL for details.
func mapToHclString(m map[string]any) string {
	keyValuePairs := []string{}

	for key, value := range m {
		keyValuePair := fmt.Sprintf(`"%s" = %s`, key, ConvertValueToHCL(value, true))
		keyValuePairs = append(keyValuePairs, keyValuePair)
	}

	return fmt.Sprintf("{%s}", strings.Join(keyValuePairs, ", "))
}

// Convert a primitive, such as a bool, int, or string, to an HCL string. If this isn't a primitive, force its value
// using Sprintf. See ConvertValueToHCL for details.
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

// Terraform allows you to pass in command-line variables using HCL syntax (e.g. -var foo=[1,2,3]). Unfortunately,
// while their golang hcl library can convert an HCL string to a Go type, they don't seem to offer a library to convert
// arbitrary Go types to an HCL string. Therefore, this method is a simple implementation that correctly handles
// ints, booleans, lists, and maps. Everything else is forced into a string using Sprintf. Hopefully, this approach is
// good enough for the type of variables we deal with in Dartboard.
func ConvertValueToHCL(value any, isNested bool) string {
	// We use type assertions to manually convert into []interface{} and map[string]interface{} if and when needed
	var v string
	var err error
	if slice, isSlice := value.([]any); isSlice {
		v = sliceToHclString(slice)
	} else if m, isMap := value.(map[string]any); isMap {
		v = mapToHclString(m)
	} else if sliceMap, isSliceMap := value.([]map[string]any); isSliceMap {
		anySlice := make([]any, len(sliceMap))
		for i, m := range sliceMap {
			anySlice[i] = m
		}
		v = sliceToHclString(anySlice)
	} else {
		v, err = primitiveToHclString(value, isNested)
	}
	if err != nil {
		log.Panicf("%v", err)
	}
	return v
}
