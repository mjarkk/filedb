package filedb

import (
	"encoding/hex"
	"errors"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

func strToHex(in string) string {
	return hex.EncodeToString([]byte(in))
}

func getHexOf(in interface{}) string {
	str := ""

	switch typeVal := in.(type) {
	case []byte:
		return hex.EncodeToString(typeVal)
	case string:
		str = typeVal
	case int:
		str = strconv.Itoa(typeVal)
	case int8:
		str = strconv.Itoa(int(typeVal))
	case int16:
		str = strconv.Itoa(int(typeVal))
	case int32:
		str = strconv.Itoa(int(typeVal))
	case int64:
		str = strconv.Itoa(int(typeVal))
	case uint:
		str = strconv.Itoa(int(typeVal))
	case uint8:
		str = strconv.Itoa(int(typeVal))
	case uint16:
		str = strconv.Itoa(int(typeVal))
	case uint32:
		str = strconv.Itoa(int(typeVal))
	case uint64:
		str = strconv.Itoa(int(typeVal))
	case bool:
		if typeVal {
			str = "true"
		} else {
			str = "false"
		}
	}

	return strToHex(str)
}

// walkThoughDir executes toExec every iteration of a dir
func walkThoughDir(dir string, kvsToMatch []string, toExec func(name string) error) error {
	files, err := ioutil.ReadDir(dir)
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, ".") {
			// We should not touch hidden files
			continue
		}
		if !keysMatchName(name, kvsToMatch) {
			continue
		}
		err := toExec(name)
		if err != nil {
			return err
		}
	}
	return err
}

// keysMatchName checks if a filename matches a list of key-values
func keysMatchName(fileName string, keyValues []string) bool {
	if strings.HasPrefix(fileName, ".") {
		return false
	}

	for _, check := range keyValues {
		if !strings.Contains(fileName, check) {
			return false
		}
	}
	return true
}

// getValueOfKey gets the value of a kv
// checkStruct needs to be ran before running this otherwhise there is a big change of a crash
func getValueOfKey(in reflect.Value, kv KV) (val interface{}, found, isZero bool) {
	keyParts := strings.Split(kv.key, ".")

	item := in
	for i, keyPart := range keyParts {
		item = item.FieldByName(keyPart)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		if !item.IsValid() {
			return nil, false, true
		}

		switch item.Kind() {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
			if !item.IsNil() {
				return nil, false, true
			}
		}

		if len(keyParts) == i+1 {
			// This is the last item
			return item.Interface(), true, item.IsZero()
		}
	}

	// This should never happen
	panic("WAIT WHAT?")
}

// setValueOfKey sets the value of a kv
// checkStruct needs to be ran before running this otherwhise there is a big change of a crash
func setValueOfKey(obj reflect.Value, kv KV) error {
	keyParts := strings.Split(kv.key, ".")

	item := obj
	for i, keyPart := range keyParts {
		item = item.FieldByName(keyPart)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		if len(keyParts) == i+1 {
			// This is the last item
			if !item.CanSet() {
				return errors.New("Cannot set value to key " + kv.key)
			}
			value := reflect.ValueOf(kv.value)
			if value.Kind() == reflect.Ptr {
				value = value.Elem()
			}

			itemKind := item.Kind()
			valueKind := value.Kind()
			if itemKind != reflect.Interface && itemKind != valueKind {
				return errors.New("cannot assign type " + valueKind.String() + " to " + itemKind.String())
			}

			item.Set(value)
			return nil
		}
	}

	// This should never happen
	panic("WAIT WHAT?")
}
