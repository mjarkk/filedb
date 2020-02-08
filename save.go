package filedb

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"
	"time"
)

// Save creates or updates a database entry
func (db *DB) Save(in interface{}) error {
	res, err := db.checkStruct(in, true, false)
	if err != nil {
		return err
	}
	value := res.reflValue

	_, isNew := getOrSetID(value)
	indexKeys := db.collections[res.objName].indexKeys

	now := createdAtKV.V(time.Now())
	if isNew {
		setValueOfKey(value, now)
	}
	setValueOfKey(value, now)

	keyName := []string{}
	addKey := func(kv KV) { keyName = append(keyName, kv.dbKey()) }

	for _, key := range indexKeys {
		keyKV := NewKV(key)
		val, _, _ := getValueOfKey(value, keyKV)
		addKey(keyKV.V(val))
	}

	bytes, err := json.Marshal(in)
	if err != nil {
		return err
	}
	if !isNew {
		// If this is a modification make sure we delete the old file first
		err := db.Delete(in)
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(path.Join(res.dbFolder, strings.Join(keyName, ":")), bytes, 0777)
}
