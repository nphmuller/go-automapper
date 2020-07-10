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

type MapOptions struct {
	UseSourceMemberList bool
}

// Map fills out the fields in dest with values from source. All fields in the
// destination object must exist in the source object.
//
// Object hierarchies with nested structs and slices are supported, as long as
// type types of nested structs/slices follow the same rules, i.e. all fields
// in destination structs must be found on the source struct.
//
// Embedded/anonymous structs are supported
//
// Values that are not exported/not public will not be mapped.
//
// It is a design decision to panic when a field cannot be mapped in the
// destination to ensure that a renamed field in either the source or
// destination does not result in subtle silent bug.
func Map(source, dest interface{}) {
	MapWithOptions(source, dest, MapOptions{})
}

// MapWithOptions fills out the fields in dest with values from source. All fields in the
// destination object must exist in the source object.
//
// Object hierarchies with nested structs and slices are supported, as long as
// type types of nested structs/slices follow the same rules, i.e. all fields
// in destination structs must be found on the source struct.
//
// Embedded/anonymous structs are supported
//
// Values that are not exported/not public will not be mapped.
//
// It is a design decision to panic when a field cannot be mapped in the
// destination to ensure that a renamed field in either the source or
// destination does not result in subtle silent bug.
func MapWithOptions(source, dest interface{}, opt MapOptions) {
	var destType = reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		panic("Dest must be a pointer type")
	}
	var sourceVal = reflect.ValueOf(source)
	var destVal = reflect.ValueOf(dest).Elem()
	mapValues(sourceVal, destVal, false, opt.UseSourceMemberList)
}

// MapLoose works just like Map, except it doesn't fail when the destination
// type contains fields not supplied by the source.
//
// This function is meant to be a temporary solution - the general idea is
// that the Map function should take a number of options that can modify its
// behavior - but I'd rather not add that functionality before I have a better
// idea what is a good options format.
func MapLoose(source, dest interface{}) {
	MapLooseWithOptions(source, dest, MapOptions{})
}

// MapLooseWithOptions works just like Map, except it doesn't fail when the destination
// type contains fields not supplied by the source.
//
// This function is meant to be a temporary solution - the general idea is
// that the Map function should take a number of options that can modify its
// behavior - but I'd rather not add that functionality before I have a better
// idea what is a good options format.
func MapLooseWithOptions(source, dest interface{}, opt MapOptions) {
	var destType = reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		panic("Dest must be a pointer type")
	}
	var sourceVal = reflect.ValueOf(source)
	var destVal = reflect.ValueOf(dest).Elem()
	mapValues(sourceVal, destVal, true, opt.UseSourceMemberList)
}

func mapValues(sourceVal, destVal reflect.Value, loose, useSourceMemberList bool) {
	sourceType := sourceVal.Type()
	destType := destVal.Type()
	if destType.Kind() == reflect.Struct && sourceVal.Type().Kind() == reflect.Ptr {
		if sourceVal.IsNil() {
			sourceVal = reflect.New(sourceType.Elem())
		}
		sourceVal = sourceVal.Elem()
		mapValues(sourceVal, destVal, loose, useSourceMemberList)
	} else if destType == sourceType {
		destVal.Set(sourceVal)
	} else if destType.Kind() == reflect.Struct && sourceType.Kind() == reflect.Struct {
		mapFields(sourceVal, destVal, loose, useSourceMemberList)
	} else if destType.Kind() == reflect.Ptr {
		if valueIsNil(sourceVal) {
			return
		}
		val := reflect.New(destType.Elem())
		mapValues(sourceVal, val.Elem(), loose, useSourceMemberList)
		destVal.Set(val)
	} else if destType.Kind() == reflect.Slice {
		mapSlice(sourceVal, destVal, loose, useSourceMemberList)
	} else {
		destVal.Set(sourceVal.Convert(destType))
	}
}

func mapSlice(sourceVal, destVal reflect.Value, loose, useSourceMemberList bool) {
	destType := destVal.Type()
	length := sourceVal.Len()
	target := reflect.MakeSlice(destType, length, length)
	for j := 0; j < length; j++ {
		val := reflect.New(destType.Elem()).Elem()
		mapValues(sourceVal.Index(j), val, loose, useSourceMemberList)
		target.Index(j).Set(val)
	}

	if length == 0 {
		verifyArrayTypesAreCompatible(sourceVal, destVal, loose, useSourceMemberList)
	}
	destVal.Set(target)
}

func verifyArrayTypesAreCompatible(sourceVal, destVal reflect.Value, loose, useSourceMemberList bool) {
	dummyDest := reflect.New(reflect.PtrTo(destVal.Type()))
	dummySource := reflect.MakeSlice(sourceVal.Type(), 1, 1)
	mapValues(dummySource, dummyDest.Elem(), loose, useSourceMemberList)
}

func mapFields(sourceVal, destVal reflect.Value, loose, useSourceMemberList bool) {
	if useSourceMemberList {
		for i := 0; i < sourceVal.NumField(); i++ {
			mapSourceField(sourceVal, destVal, i, loose, useSourceMemberList)
		}
	} else {
		for i := 0; i < destVal.NumField(); i++ {
			mapDestField(sourceVal, destVal, i, loose, useSourceMemberList)
		}
	}
}

func mapDestField(source, destVal reflect.Value, i int, loose, useSourceMemberList bool) {
	destType := destVal.Type()
	destTypeField := destType.Field(i)
	fieldName := destTypeField.Name
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Error mapping field: %s. DestType: %v. SourceType: %v. Error: %v", fieldName, destType, source.Type(), r))
		}
	}()

	if automapperTag, ok := destTypeField.Tag.Lookup("automapper"); ok && automapperTag == "-" {
		return
	}

	destField := destVal.Field(i)
	if destType.Field(i).Anonymous {
		mapValues(source, destField, loose, useSourceMemberList)
	} else {
		if valueIsContainedInNilEmbeddedType(source, fieldName) {
			return
		}
		sourceField := source.FieldByName(fieldName)
		if (sourceField == reflect.Value{}) {
			if loose {
				return
			}
			if destField.Kind() == reflect.Struct {
				mapValues(source, destField, loose, useSourceMemberList)
				return
			} else {
				for i := 0; i < source.NumField(); i++ {
					if source.Field(i).Kind() != reflect.Struct {
						continue
					}
					if sourceField = source.Field(i).FieldByName(fieldName); (sourceField != reflect.Value{}) {
						break
					}
				}
			}
		}
		mapValues(sourceField, destField, loose, useSourceMemberList)
	}
}
func mapSourceField(source, destVal reflect.Value, i int, loose, useSourceMemberList bool) {
	sourceType := source.Type()
	sourceTypeField := sourceType.Field(i)
	fieldName := sourceTypeField.Name
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Error mapping field: %s. DestType: %v. SourceType: %v. Error: %v", fieldName, destVal.Type(), sourceType, r))
		}
	}()

	sourceFieldName := source.Type().Field(i).Name
	for q := 0; q < destVal.Type().NumField(); q++ {
		destFieldName := destVal.Type().Field(q).Name
		if sourceFieldName == destFieldName {
			mapDestField(source, destVal, q, loose, useSourceMemberList)
			return
		}
	}
	panic("destination has no field that matches source field")
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
