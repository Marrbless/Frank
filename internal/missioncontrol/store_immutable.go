package missioncontrol

import (
	"errors"
	"os"
	"reflect"
)

func storeImmutableJSONRecord[T any](path string, record T, load func(string) (T, error), duplicateErr func() error) (T, bool, error) {
	var zero T
	existing, err := load(path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return zero, false, duplicateErr()
	}
	if !errors.Is(err, os.ErrNotExist) {
		return zero, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return zero, false, err
	}
	stored, err := load(path)
	if err != nil {
		return zero, false, err
	}
	return stored, true, nil
}
