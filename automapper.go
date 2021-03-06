// Copyright (c) 2015 Peter Strøiman, distributed under the MIT license

// Package automapper provides support for mapping between two different types
// with compatible fields. The intended application for this is when you use
// one set of types to represent DTOs (data transfer objects, e.g. json data),
// and a different set of types internally in the application. Using this
// package can help converting from one type to another.
//
// This package uses reflection to perform mapping which should be fine for
// all but the most demanding applications.
package automapper

import (
	"fmt"
	"reflect"
)

type mapOptions struct {
	useSourceMemberList bool
}

// MapToDestination fills out the fields in dest with values from source. All fields in the
// destination object must exist in the source object.
func MapToDestination(source, dest interface{}) {
	var destType = reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		panic("Dest must be a pointer type")
	}
	var sourceVal = reflect.ValueOf(source)
	var destVal = reflect.ValueOf(dest).Elem()
	mapValues(sourceVal, destVal, mapOptions{useSourceMemberList: false})
}

// MapFromSource fills out the fields in dest with values from source. All fields in the
// source object must exist in the destination object.
func MapFromSource(source, dest interface{}) {
	var destType = reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		panic("Dest must be a pointer type")
	}
	var sourceVal = reflect.ValueOf(source)
	var destVal = reflect.ValueOf(dest).Elem()
	mapValues(sourceVal, destVal, mapOptions{useSourceMemberList: true})
}

// MapFromSourceMap fills out the fields in dest with values from source map. All fields in the
// source map must exist in the destination object.
func MapFromSourceMap(source map[string]interface{}, dest interface{}) {
	var destType = reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		panic("Dest must be a pointer type")
	}

	var destVal = reflect.ValueOf(dest).Elem()
	for key, value := range source {
		destFieldVal := destVal.FieldByName(key)
		mapValues(reflect.ValueOf(value), destFieldVal, mapOptions{useSourceMemberList: true})
	}
}

func mapValues(sourceVal, destVal reflect.Value, opts mapOptions) {
	sourceType := sourceVal.Type()
	destType := destVal.Type()
	if destType.Kind() == reflect.Struct && sourceVal.Type().Kind() == reflect.Ptr {
		if sourceVal.IsNil() {
			sourceVal = reflect.New(sourceType.Elem())
		}
		sourceVal = sourceVal.Elem()
		mapValues(sourceVal, destVal, opts)
	} else if destType == sourceType {
		destVal.Set(sourceVal)
	} else if destType.Kind() == reflect.Struct && sourceType.Kind() == reflect.Struct {
		mapFields(sourceVal, destVal, opts)
	} else if destType.Kind() == reflect.Ptr {
		if valueIsNil(sourceVal) {
			return
		}
		val := reflect.New(destType.Elem())
		mapValues(sourceVal, val.Elem(), opts)
		destVal.Set(val)
	} else if destType.Kind() == reflect.Slice {
		mapSlice(sourceVal, destVal, opts)
	} else {
		destVal.Set(sourceVal.Convert(destType))
	}
}

func mapSlice(sourceVal, destVal reflect.Value, opts mapOptions) {
	destType := destVal.Type()
	length := sourceVal.Len()
	target := reflect.MakeSlice(destType, length, length)
	for j := 0; j < length; j++ {
		val := reflect.New(destType.Elem()).Elem()
		mapValues(sourceVal.Index(j), val, opts)
		target.Index(j).Set(val)
	}

	if length == 0 {
		verifyArrayTypesAreCompatible(sourceVal, destVal, opts)
	}
	destVal.Set(target)
}

func verifyArrayTypesAreCompatible(sourceVal, destVal reflect.Value, opts mapOptions) {
	dummyDest := reflect.New(reflect.PtrTo(destVal.Type()))
	dummySource := reflect.MakeSlice(sourceVal.Type(), 1, 1)
	mapValues(dummySource, dummyDest.Elem(), opts)
}

func mapFields(sourceVal, destVal reflect.Value, opts mapOptions) {
	if opts.useSourceMemberList {
		for i := 0; i < sourceVal.NumField(); i++ {
			mapSourceField(sourceVal, destVal, i, opts)
		}
	} else {
		for i := 0; i < destVal.NumField(); i++ {
			mapDestField(sourceVal, destVal, i, opts)
		}
	}
}

func mapDestField(source, destVal reflect.Value, i int, opts mapOptions) {
	destType := destVal.Type()
	destTypeField := destType.Field(i)
	destFieldName := destTypeField.Name
	sourceFieldName := destFieldName

	if automapperTag, ok := destTypeField.Tag.Lookup("automapper"); ok {
		if automapperTag == "-" {
			return
		}
		sourceFieldName = automapperTag
	}

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Error mapping field: %s. DestType: %v. SourceType: %v. Error: %v", destFieldName, destType, source.Type(), r))
		}
	}()

	destField := destVal.Field(i)
	if destType.Field(i).Anonymous {
		mapValues(source, destField, opts)
	} else {
		mapByFieldName(source, destVal, opts, sourceFieldName, destFieldName)
	}
}

func mapSourceField(source, destVal reflect.Value, i int, opts mapOptions) {
	sourceType := source.Type()
	sourceTypeField := sourceType.Field(i)
	sourceFieldName := sourceTypeField.Name
	destFieldName := sourceFieldName

	if automapperTag, ok := sourceTypeField.Tag.Lookup("automapper"); ok {
		if automapperTag == "-" {
			return
		}
		destFieldName = automapperTag
	}

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Error mapping field: %s. DestType: %v. SourceType: %v. Error: %v", sourceFieldName, destVal.Type(), sourceType, r))
		}
	}()

	sourceField := source.Field(i)
	if sourceType.Field(i).Anonymous {
		mapValues(sourceField, destVal, opts)
	} else {
		mapByFieldName(source, destVal, opts, sourceFieldName, destFieldName)
	}
}

func mapByFieldName(source, destVal reflect.Value, opts mapOptions, sourceFieldName, destFieldName string) {
	destField := destVal.FieldByName(destFieldName)
	if valueIsContainedInNilEmbeddedType(source, sourceFieldName) {
		return
	}
	sourceField := source.FieldByName(sourceFieldName)
	if (sourceField == reflect.Value{}) {
		if destField.Kind() == reflect.Struct {
			mapValues(source, destField, opts)
			return
		} else {
			for i := 0; i < source.NumField(); i++ {
				if source.Field(i).Kind() != reflect.Struct {
					continue
				}
				if sourceField = source.Field(i).FieldByName(sourceFieldName); (sourceField != reflect.Value{}) {
					break
				}
			}
		}
	}
	mapValues(sourceField, destField, opts)
}

func valueIsNil(value reflect.Value) bool {
	return value.Type().Kind() == reflect.Ptr && value.IsNil()
}

func valueIsContainedInNilEmbeddedType(source reflect.Value, fieldName string) bool {
	structField, _ := source.Type().FieldByName(fieldName)
	ix := structField.Index
	if len(structField.Index) > 1 {
		parentField := source.FieldByIndex(ix[:len(ix)-1])
		if valueIsNil(parentField) {
			return true
		}
	}
	return false
}
