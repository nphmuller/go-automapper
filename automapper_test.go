// Copyright (c) 2015 Peter Strøiman, distributed under the MIT license

package automapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPanicWhenDestIsNotPointer(t *testing.T) {
	defer func() { recover() }()
	source, dest := SourceTypeA{}, DestTypeA{}
	MapToDestination(source, dest)

	t.Error("Should have panicked")
}

func TestDestinationIsUpdatedFromSource(t *testing.T) {
	source, dest := SourceTypeA{Foo: 42}, DestTypeA{}
	MapToDestination(source, &dest)
	assert.Equal(t, 42, dest.Foo)
}

func TestDestinationIsUpdatedFromSourceWhenSourcePassedAsPtr(t *testing.T) {
	source, dest := SourceTypeA{42, "Bar"}, DestTypeA{}
	MapToDestination(&source, &dest)
	assert.Equal(t, 42, dest.Foo)
	assert.Equal(t, "Bar", dest.Bar)
}

func TestWithNestedTypes(t *testing.T) {
	source := struct {
		Baz   string
		Child SourceTypeA
	}{}
	dest := struct {
		Baz   string
		Child DestTypeA
	}{}

	source.Baz = "Baz"
	source.Child.Bar = "Bar"
	MapToDestination(&source, &dest)
	assert.Equal(t, "Baz", dest.Baz)
	assert.Equal(t, "Bar", dest.Child.Bar)
}

func TestWithSourceSecondLevel(t *testing.T) {
	source := struct {
		Child DestTypeA
	}{}
	dest := SourceTypeA{}

	source.Child.Bar = "Bar"
	MapToDestination(&source, &dest)
	assert.Equal(t, "Bar", dest.Bar)
}

func TestWithDestSecondLevel(t *testing.T) {
	source := SourceTypeA{}
	dest := struct {
		Child DestTypeA
	}{}

	source.Bar = "Bar"
	MapToDestination(&source, &dest)
	assert.Equal(t, "Bar", dest.Child.Bar)
}

func TestWithSliceTypes(t *testing.T) {
	source := struct {
		Children []SourceTypeA
	}{}
	dest := struct {
		Children []DestTypeA
	}{}
	source.Children = []SourceTypeA{
		SourceTypeA{Foo: 1},
		SourceTypeA{Foo: 2}}

	MapToDestination(&source, &dest)
	assert.Equal(t, 1, dest.Children[0].Foo)
	assert.Equal(t, 2, dest.Children[1].Foo)
}

func TestWithMultiLevelSlices(t *testing.T) {
	source := struct {
		Parents []SourceParent
	}{}
	dest := struct {
		Parents []DestParent
	}{}
	source.Parents = []SourceParent{
		SourceParent{
			Children: []SourceTypeA{
				SourceTypeA{Foo: 42},
				SourceTypeA{Foo: 43},
			},
		},
		SourceParent{
			Children: []SourceTypeA{},
		},
	}

	MapToDestination(&source, &dest)
}

func TestWithEmptySliceAndIncompatibleTypes(t *testing.T) {
	defer func() { recover() }()

	source := struct {
		Children []struct{ Foo string }
	}{}
	dest := struct {
		Children []struct{ Bar int }
	}{}

	MapToDestination(&source, &dest)
	t.Error("Should have panicked")
}

func TestWhenSourceIsMissingField(t *testing.T) {
	defer func() { recover() }()
	source := struct {
		A string
	}{}
	dest := struct {
		A, B string
	}{}
	MapToDestination(&source, &dest)
	t.Error("Should have panicked")
}

func TestWithUnnamedFields(t *testing.T) {
	source := struct {
		Baz string
		SourceTypeA
	}{}
	dest := struct {
		Baz string
		DestTypeA
	}{}
	source.Baz = "Baz"
	source.SourceTypeA.Foo = 42

	MapToDestination(&source, &dest)
	assert.Equal(t, "Baz", dest.Baz)
	assert.Equal(t, 42, dest.DestTypeA.Foo)
}

func TestWithPointerFieldsNotNil(t *testing.T) {
	source := struct {
		Foo *SourceTypeA
	}{}
	dest := struct {
		Foo *DestTypeA
	}{}
	source.Foo = nil

	MapToDestination(&source, &dest)
	assert.Nil(t, dest.Foo)
}

func TestWithPointerFieldsNil(t *testing.T) {
	source := struct {
		Foo *SourceTypeA
	}{}
	dest := struct {
		Foo *DestTypeA
	}{}
	source.Foo = &SourceTypeA{Foo: 42}

	MapToDestination(&source, &dest)
	assert.NotNil(t, dest.Foo)
	assert.Equal(t, 42, dest.Foo.Foo)
}

func TestMapToDestinationPointerToNonPointerTypeWithData(t *testing.T) {
	source := struct {
		Foo *SourceTypeA
	}{}
	dest := struct {
		Foo DestTypeA
	}{}
	source.Foo = &SourceTypeA{Foo: 42}

	MapToDestination(&source, &dest)
	assert.NotNil(t, dest.Foo)
	assert.Equal(t, 42, dest.Foo.Foo)
}

func TestMapToDestinationPointerToNonPointerTypeWithoutData(t *testing.T) {
	source := struct {
		Foo *SourceTypeA
	}{}
	dest := struct {
		Foo DestTypeA
	}{}
	source.Foo = nil

	MapToDestination(&source, &dest)
	assert.NotNil(t, dest.Foo)
	assert.Equal(t, 0, dest.Foo.Foo)
}

func TestMapToDestinationPointerToAnonymousTypeToFieldName(t *testing.T) {
	source := struct {
		*SourceTypeA
	}{}
	dest := struct {
		Foo int
	}{}
	source.SourceTypeA = nil

	MapToDestination(&source, &dest)
	assert.Equal(t, 0, dest.Foo)
}

func TestMapToDestinationPointerToNonPointerTypeWithoutDataAndIncompatibleType(t *testing.T) {
	defer func() { recover() }()
	// Just make sure we stil panic
	source := struct {
		Foo *SourceTypeA
	}{}
	dest := struct {
		Foo struct {
			Baz string
		}
	}{}
	source.Foo = nil

	MapToDestination(&source, &dest)
	t.Error("Should have panicked")
}

func TestWhenUsingIncompatibleTypes(t *testing.T) {
	defer func() { recover() }()
	source := struct{ Foo string }{}
	dest := struct{ Foo int }{}
	MapToDestination(&source, &dest)
	t.Error("Should have panicked")
}

func TestSetStructOfSameTypeDirectly(t *testing.T) {
	type FooType struct {
		time.Time
	}
	source := struct {
		Foo FooType
	}{FooType{Time: time.Now().UTC()}}
	dest := struct {
		Foo FooType
	}{}
	MapToDestination(&source, &dest)
	assert.Equal(t, source.Foo.String(), dest.Foo.String())
}

func TestNamedType(t *testing.T) {
	type SourceType string
	type DestType string
	source := struct {
		Foo SourceType
	}{"abc"}
	dest := struct {
		Foo DestType
	}{}
	MapToDestination(&source, &dest)
	assert.Equal(t, string(source.Foo), string(dest.Foo))
}

func TestSkip(t *testing.T) {
	source := struct {
		Foo string
	}{"abc"}
	dest := struct {
		Foo string
		Bar string `automapper:"-"`
	}{}
	MapToDestination(&source, &dest)
	assert.Empty(t, dest.Bar)
}

func TestMapToDestination(t *testing.T) {
	source := struct {
		Foo string
	}{"abc"}
	dest := struct {
		Bar string `automapper:"Foo"`
	}{}
	MapToDestination(&source, &dest)
	assert.Equal(t, source.Foo, dest.Bar)
}

func TestMapFromSource(t *testing.T) {
	source := struct {
		Foo string `automapper:"Bar"`
	}{"abc"}
	dest := struct {
		Bar string
	}{}
	MapFromSource(&source, &dest)
	assert.Equal(t, source.Foo, dest.Bar)
}

func TestMapSourceField_DestContainsUnmappedFields(t *testing.T) {
	source := struct {
		Foo string
	}{"abc"}
	dest := struct {
		Foo string
		Bar string
	}{}
	MapFromSource(&source, &dest)
	assert.Equal(t, source.Foo, dest.Foo)
}

func TestMapSourceField_Panics(t *testing.T) {
	defer func() { recover() }()
	source := struct {
		Foo string
	}{"abc"}
	dest := struct {
	}{}
	MapFromSource(&source, &dest)
	t.Error("Should have panicked")
}

func TestMapSourceField_Skip(t *testing.T) {
	source := struct {
		Foo string `automapper:"-"`
	}{"abc"}
	dest := struct {
		Bar string
	}{}
	MapFromSource(&source, &dest)
	assert.Empty(t, dest.Bar)
}

func TestMapSourceField_FromAnonymous(t *testing.T) {
	source := struct{
		SourceTypeA
	}{
		SourceTypeA: SourceTypeA{Foo: 42},
	}
	dest := DestTypeA{}
	MapFromSource(&source, &dest)
	assert.Equal(t, source.Foo, dest.Foo)
}

func TestMapSourceField_ToAnonymous(t *testing.T) {
	source := SourceTypeA{Foo: 42}
	dest := struct{
		DestTypeA
	}{}
	MapFromSource(&source, &dest)
	assert.Equal(t, source.Foo, dest.Foo)
}

func TestMapSourceField_BothAnonymous(t *testing.T) {
	source := struct {
		SourceTypeA
	}{
		SourceTypeA: SourceTypeA{Foo: 42},
	}
	dest := struct {
		DestTypeA
	}{}

	MapFromSource(&source, &dest)
	assert.Equal(t, source.Foo, dest.Foo)
}

func TestMapFromSourceMap(t *testing.T) {
	type childSrc struct {
		Foo string
	}
	type childDest struct {
		Foo string
		Bar string
	}
	source := map[string]interface{}{
		"Foo": "abc",
		"Child": childSrc{Foo: "456"},
	}
	dest := struct {
		Foo string
		Bar string
		Child childDest
	}{Bar: "123"}

	MapFromSourceMap(source, &dest)

	assert.Equal(t, "abc", dest.Foo, "should map direct field")
	assert.Equal(t, "123", dest.Bar, "field should not be overwritten")
	assert.Equal(t, "456", dest.Child.Foo, "struct fields should be mapped")
}

type SourceParent struct {
	Children []SourceTypeA
}

type DestParent struct {
	Children []DestTypeA
}

type SourceTypeA struct {
	Foo int
	Bar string
}

type DestTypeA struct {
	Foo int
	Bar string
}
