package utils_test

import (
	"testing"

	"github.com/maxime915/glauncher/utils"
	"github.com/stretchr/testify/assert"
)

// a struct with fields
type Struct1 struct {
	FieldString string         `json:"field-string"`
	FieldInt    int            `json:"field-int"`
	FieldBool   bool           `json:"field-bool"`
	FieldSlice  []int          `json:"field-slice"`
	FieldMap    map[string]int `json:"field-map"`
}

func TestNilValues(t *testing.T) {
	_, err := utils.ValFromJSON[Struct1](nil)
	assert.NoError(t, err)

	s := Struct1{}
	err = utils.FromJSON(nil, &s)
	assert.NoError(t, err)

	s2 := &Struct1{}
	err = utils.FromJSON(nil, s2)
	assert.NoError(t, err)

	err = utils.FromJSON(nil, &s2)
	assert.NoError(t, err)
}

func TestConservation(t *testing.T) {
	// All fields are set
	s := Struct1{
		FieldString: "string",
		FieldInt:    42,
		FieldBool:   true,
		FieldSlice:  []int{1, 2, 3},
		FieldMap:    map[string]int{"a": 1, "b": 2},
	}

	// Convert to JSON
	json, err := utils.ValToJSON(s)
	assert.NoError(t, err)

	// Convert back to struct
	s2, err := utils.ValFromJSON[Struct1](json)
	assert.NoError(t, err)

	// Check that the values are the same
	assert.Equal(t, s, s2)
}

func TestProducingEmptyFields(t *testing.T) {
	// All fields are set
	s := Struct1{
		FieldString: "string",
		FieldInt:    42,
		FieldBool:   true,
		FieldSlice:  []int{1, 2, 3},
		FieldMap:    map[string]int{"a": 1, "b": 2},
	}

	// Convert to JSON
	json, err := utils.ValToJSON(s)
	assert.NoError(t, err)

	// remove some fields
	delete(json, "field-string")
	delete(json, "field-slice")
	delete(json, "field-map")

	// Convert back to struct
	s2, err := utils.ValFromJSON[Struct1](json)
	assert.NoError(t, err)

	// Check that the values are the same
	assert.Equal(t, s2.FieldString, "")
	assert.Equal(t, s2.FieldInt, 42)
	assert.Equal(t, s2.FieldBool, true)
	assert.Nil(t, s2.FieldSlice)
	assert.Nil(t, s2.FieldMap)
}
