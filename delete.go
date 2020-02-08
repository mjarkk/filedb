package filedb

import (
	"errors"
	"os"
	"path"
)

// Delete removes an item from the database
//
// NOTE: It's currently not possible to delete a slice of items though
//       it will delete ALL items matching a set struct field expect
//       if ID is set then only 1 will be removed and all
//       other fields will be ignored
//
// NOTE: If the input struct has a filledin M.ID all other fields will
//       be ignored
func (db *DB) Delete(in interface{}) error {
	res, err := db.checkStruct(in, true, false)
	if err != nil {
		return err
	}

	searchFor := []string{}

	indexKeys := db.collections[res.objName].indexKeys
	onlySearchingForID := false

	for _, key := range indexKeys {
		val, _, iszero := getValueOfKey(res.reflValue, NewKV(key))
		if !iszero {
			kv := KV{key, val}.dbKey()
			if key == IDkv.key {
				searchFor = []string{kv}
				onlySearchingForID = true
				break
			}
			searchFor = append(searchFor, kv)
		}
	}

	stoppedForOnlyID := errors.New("stopped for only id")
	err = walkThoughDir(res.dbFolder, searchFor, func(name string) error {
		err = os.Remove(path.Join(res.dbFolder, name))
		if err != nil {
			return err
		}
		if onlySearchingForID {
			return stoppedForOnlyID
		}
		return nil
	})
	if err != nil && err != stoppedForOnlyID {
		return err
	}

	return nil
}
