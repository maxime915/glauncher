package entry

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/maxime915/glauncher/version"
)

var (
	registeredTypes          = make(map[string]reflect.Type)
	ErrTypeAlreadyRegistered = errors.New("type already registered")
	ErrTypeNotRegistered     = errors.New("type not registered")
	ErrVersionMisMatch       = errors.New("serialized entry was built with a different version of the launcher")
)

// return a unique identifier for a type
func typeKey(type_ reflect.Type) string {
	return type_.PkgPath() + "." + type_.Name()
}

// Register a type to be serialized
func RegisterEntryType[T Entry]() error {
	var entry T
	entryType := reflect.TypeOf(entry)
	entryTypeKey := typeKey(entryType)
	if _, ok := registeredTypes[entryTypeKey]; ok {
		return ErrTypeAlreadyRegistered
	}
	registeredTypes[entryTypeKey] = entryType
	return nil
}

type serialization struct {
	Type         string            `json:"type"`
	Data         []byte            `json:"data"`
	Options      map[string]string `json:"options"`
	BuildVersion string            `json:"build_version"`
}

// Serialize an entry to a byte slice.
// NOTE: the then de-serialized entry will be a pointer type.
// NOTE: the type of the entry MUST be registered beforehand (see RegisterEntryType[T]()).
func Serialize(entry Entry) ([]byte, error) {
	return SerializeWithOptions(entry, nil)
}

func SerializeWithOptions(entry Entry, options map[string]string) ([]byte, error) {
	var serialized serialization
	var err error

	// store (registered) type
	serialized.Type = typeKey(reflect.TypeOf(entry))
	if _, ok := registeredTypes[serialized.Type]; !ok {
		return nil, ErrTypeNotRegistered
	}

	// store data
	serialized.Data, err = json.Marshal(entry)
	if err != nil {
		return nil, err
	}

	// store options
	serialized.Options = options

	// versioning
	serialized.BuildVersion = version.BuildVersion()

	return json.Marshal(serialized)
}

// Return a new entry from the serialization
// NOTE: the de-serialized entry will be a pointer type.
// NOTE: the type of the entry MUST be registered beforehand (see RegisterEntryType[T]())
func Deserialize(data []byte) (Entry, error) {
	entry, _, err := DeserializeWithOption(data)
	return entry, err
}

func DeserializeWithOption(data []byte) (entry Entry, options map[string]string, err error) {
	var serialized serialization
	err = json.Unmarshal(data, &serialized)
	if err != nil {
		return nil, nil, err
	}

	if serialized.BuildVersion != version.BuildVersion() {
		return nil, nil, ErrVersionMisMatch
	}

	// load (registered) type
	entryType, ok := registeredTypes[serialized.Type]
	if !ok {
		return nil, nil, ErrTypeNotRegistered
	}

	// create a ptr to store the deserialized data
	entry, ok = reflect.New(entryType).Interface().(Entry)
	if !ok {
		return nil, nil, ErrTypeNotRegistered
	}

	// deserialize entry
	err = json.Unmarshal(serialized.Data, entry)
	if err != nil {
		return nil, nil, err
	}

	return entry, serialized.Options, nil
}
