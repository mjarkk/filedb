package filedb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
)

// WhiteListEntry is one item to whitelist in the WhiteListGroup function
type WhiteListEntry struct {
	Obj        interface{}
	SearchKeys []KV
}

// WhiteListGroup whitelists multiple items
// Handy if you have a lot of structs to whitelist
func (db *DB) WhiteListGroup(toWhitelist []WhiteListEntry) error {
	if toWhitelist == nil {
		return nil
	}
	for _, item := range toWhitelist {
		searchKeys := []KV{}
		if item.SearchKeys != nil {
			searchKeys = item.SearchKeys
		}
		err := db.Whitelist(item.Obj, searchKeys...)
		if err != nil {
			return err
		}
	}
	return nil
}

// Whitelist Whitelists a struct to be allowed inside the database
// This also adds searchKeys so the struct can be searched for,
// M.ID is automatily always added as search key
func (db *DB) Whitelist(in interface{}, searchKeys ...KV) error {
	res, err := db.checkStruct(in, false, false)
	if err != nil {
		return err
	}
	val := res.reflValue

	for _, kv := range searchKeys {
		key := kv.key

		switch key {
		case "":
			return errors.New("empty keys are not allowed")
		case "M.ID":
			return errors.New("M.ID is added by default and thus should not be added manually to .Index search keys")
		}

		keyParts := strings.Split(key, ".")
		currentVal := val
		for i, keyPart := range keyParts {

			if i+1 != len(keyParts) {
				// go into a nested value
				currentVal = val.FieldByName(keyPart)
				currentValKind := currentVal.Kind()
				if currentValKind == reflect.Invalid {
					return errors.New("part " + keyPart + " not found in key " + key)
				}
				if currentValKind == reflect.Ptr {
					currentVal = currentVal.Elem()
					currentValKind = currentVal.Kind()
				}
				if currentValKind != reflect.Struct {
					return errors.New("part " + keyPart + " of key " + key + " is not a valid, for nested types only structs and ptr's to structs are allowed")
				}
				continue
			}

			keyField := currentVal.FieldByName(keyPart)
			if keyField.Kind() == reflect.Invalid {
				return errors.New("key " + key + " not found in struct")
			}

			keyType := keyField.Type()
			kind := keyType.Kind()
			if kind == reflect.Ptr {
				keyType = keyType.Elem()
				kind = keyType.Kind()
			}

			allowed := []reflect.Kind{
				reflect.String,
				reflect.Bool,
				reflect.Int,
				reflect.Int8,
				reflect.Int16,
				reflect.Int32,
				reflect.Int64,
				reflect.Uint,
				reflect.Uint8,
				reflect.Uint16,
				reflect.Uint32,
				reflect.Uint64,
			}

			isAllowed := false
			for _, allowedKind := range allowed {
				isAllowed = allowedKind == kind
				if isAllowed {
					break
				}
			}
			if !isAllowed {
				return errors.New("Key type (" + keyType.String() + ") is not allowed, allowed types are string, bool, (u)int(8,16,32,64)")
			}
		}
	}

	err = os.MkdirAll(res.dbFolder, 0777)
	if err != nil {
		return err
	}

	indexKeys := make([]string, len(searchKeys))
	for i, kv := range searchKeys {
		indexKeys[i] = kv.key
	}
	indexKeys = append(indexKeys, "M.ID") // Add M.ID by default

	db.collections[res.objName] = collection{
		indexKeys: indexKeys,
		folder:    res.dbFolder,
	}

	infoBytes, err := ioutil.ReadFile(path.Join(res.dbFolder, ".info"))
	if err == nil {
		var oldInfo collectionInfo
		err = json.Unmarshal(infoBytes, &oldInfo)
		if err == nil {
			// compair the old to the new database names and check if we need to migrate something
			if strings.Join(indexKeys, "+") != strings.Join(oldInfo.IndexableKeys, "+") {
				fmt.Println("The last used KVs don't matchup with the new KVs, fixing the naming")

			}
		}
	}

	infoBytes, err = json.Marshal(collectionInfo{IndexableKeys: indexKeys})
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(res.dbFolder, ".info"), infoBytes, 0777)
	if err != nil {
		return err
	}

	return nil
}
