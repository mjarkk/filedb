package filedb

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

// Find finds entries in the database based on the KVs (key values) provided
func (db *DB) Find(bindTo interface{}, keyValues ...KV) error {
	res, err := db.checkStruct(bindTo, true, true)
	if err != nil {
		return err
	}

	keysToCheck := []string{}
	for _, kv := range keyValues {
		if kv.value == "" {
			continue
		}
		keysToCheck = append(keysToCheck, kv.dbKey())
	}

	// It's not possible to really walk though a dir, every walk function seems to request the full folder contents, then walk though it
	// There doesn't seem to be an easy way to walk though a dir without requesting the full contents all at once
	contents, err := ioutil.ReadDir(res.dbFolder)
	if err != nil {
		return err
	}

	if !res.isSlice {
		for _, item := range contents {
			name := item.Name()
			if !keysMatchName(name, keysToCheck) {
				continue
			}
			jsonBytes, err := ioutil.ReadFile(path.Join(res.dbFolder, name))
			if err != nil {
				return err
			}
			return json.Unmarshal(jsonBytes, bindTo)
		}
		return ErrNoDocumentFound
	}

	var wg sync.WaitGroup
	wg.Add(len(contents))
	buff := bytes.NewBuffer([]byte("["))
	var buffLock sync.RWMutex

	for _, item := range contents {
		go func(item os.FileInfo) {
			defer wg.Done()

			name := item.Name()
			if !keysMatchName(name, keysToCheck) {
				return
			}

			jsonBytes, err := ioutil.ReadFile(path.Join(res.dbFolder, name))
			if err != nil {
				return
			}

			if buff.Len() != 1 {
				joinedjsonBytes := make([]byte, 1+len(jsonBytes))
				joinedjsonBytes[0] = ','
				copy(joinedjsonBytes[1:], jsonBytes)
				jsonBytes = joinedjsonBytes
			}
			buffLock.Lock()
			buff.Write(jsonBytes)
			buffLock.Unlock()
		}(item)
	}

	wg.Wait()
	buff.Write([]byte("]"))

	jsonDecoder := json.NewDecoder(buff)
	err = jsonDecoder.Decode(bindTo)
	if err != nil {
		return err
	}
	buff.Reset()

	return nil
}
