/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 */

package utils

import (
	"bytes"
	"errors"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// MarshalK8SResources marshals every field of a struct into a k8s resource YAML.
func MarshalK8SResources(resources interface{}) ([]byte, error) {
	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	var buf bytes.Buffer

	// reflect over struct containing fields that are k8s resources
	value := reflect.ValueOf(resources)
	if value.Kind() != reflect.Ptr && value.Kind() != reflect.Interface {
		return nil, errors.New("marshal on non-pointer called")
	}
	elem := value.Elem()
	if elem.Kind() == reflect.Struct {
		// iterate over all struct fields
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Field(i)
			var inter interface{}
			// check if value can be converted to interface
			if field.CanInterface() {
				inter = field.Addr().Interface()
			} else {
				continue
			}
			// convert field interface to runtime.Object
			obj, ok := inter.(runtime.Object)
			if !ok {
				continue
			}

			if i > 0 {
				// separate YAML documents
				buf.Write([]byte("---\n"))
			}
			// serialize k8s resource
			if err := serializer.Encode(obj, &buf); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}
