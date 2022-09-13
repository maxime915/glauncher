package utils

import "encoding/json"

// FromJSON updates val with the information from source, corresponding to a JSON
// encoding of a struct of type T.
// NOTE: absent slice and map fields are set to nil, not to empty slices or maps.
func FromJSON[T any](source map[string]any, val *T) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, val)
}

// ValFromJSON builds a struct from a map[string]any, corresponding to JSON encoding
// of the struct.
// NOTE: absent slice and map fields are set to nil, not to empty slices or maps.
func ValFromJSON[T any](source map[string]any) (T, error) {
	var target T
	err := FromJSON(source, &target)
	return target, err
}

// ValToJSON builds a map[string]any from a struct, corresponding to JSON encoding
func ValToJSON[T any](source T) (map[string]any, error) {
	var target map[string]any

	data, err := json.Marshal(source)
	if err != nil {
		return target, err
	}

	err = json.Unmarshal(data, &target)
	return target, err
}
