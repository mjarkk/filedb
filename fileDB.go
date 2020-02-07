package filedb

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

func strToHex(in string) string {
	return hex.EncodeToString([]byte(in))
}

// M needs to be inside a struct for it to be a valid
type M struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var (
	// IDkv is the key value for M.ID
	IDkv = KV{key: "M.ID"}

	// createdAtKV contains the key for the created at in M
	createdAtKV = KV{key: "M.CreatedAt"}

	// updatedAtKV contains the key for the updated at in M
	updatedAtKV = KV{key: "M.UpdatedAt"}

	// ErrNoDocumentFound explains itself mostly
	ErrNoDocumentFound = errors.New("Did not found requested document")
)

// DB contains the database information
type DB struct {
	rootDir     string
	collections map[string]collection // a map of all allowed struct in the database
}

// NewKV creates a new key value store where k is the key
func NewKV(k string) KV {
	return KV{key: k}
}

// KV is a key value store
type KV struct {
	key   string
	value interface{}
}

// V returns the keyvalue instace with a value
func (k KV) V(v interface{}) KV {
	k.value = v
	return k
}

// dbKey returns the a part of the stored file
func (k KV) dbKey() string {
	return k.key + "=" + getHexOf(k.value) + "=" + k.key
}

type collection struct {
	indexKeys []string
	folder    string
}

// location returns the full location of a file
func (db *DB) location(add ...string) string {
	return path.Join(append([]string{db.rootDir}, add...)...)
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
			if buff.Len() == 1 {
				buff.Write(jsonBytes)
			} else {
				joinedjsonBytes := make([]byte, 1+len(jsonBytes))
				joinedjsonBytes[0] = ','
				copy(joinedjsonBytes[1:], jsonBytes)
				buff.Write(joinedjsonBytes)
			}
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

func keysMatchName(fileName string, keyValues []string) bool {
	for _, check := range keyValues {
		if !strings.Contains(fileName, check) {
			return false
		}
	}
	return true
}

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

type collectionResolver struct {
	reflValue         reflect.Value // NOTE: if checkStruct is ran with resolverSlice true this value can't probebly be trusted!!, use reflType instaid
	reflType          reflect.Type
	objName, dbFolder string
	isSlice           bool
}

// getID returns the id of a object
// checkStruct needs to be ran before running this otherwhise there is a big change of a crash
func getID(in reflect.Value) string {
	val, _, _ := getValueOfKey(in, IDkv)
	return val.(string)
}

// getOrSetID returns the id of the object and if it doesn't have one yet it adds an id
// checkStruct needs to be ran before running this otherwhise there is a big change of a crash
func getOrSetID(in reflect.Value) (id string, setted bool) {
	id = getID(in)
	if id != "" {
		return id, false
	}
	newID := uuid.NewV4().String()

	setValueOfKey(in, IDkv.V(newID))
	return newID, true
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

func (db *DB) checkStruct(in interface{}, checkCollection, resolverSlice bool) (*collectionResolver, error) {
	isSlice := false
	inValue := reflect.ValueOf(in)
	inType := inValue.Type()

	if inType.Kind() == reflect.Ptr {
		inValue = inValue.Elem()
		inType = inValue.Type()
	}

	if resolverSlice && inType.Kind() == reflect.Slice {
		inType = inType.Elem()
		isSlice = true
		if inType.Kind() == reflect.Ptr {
			inType = inType.Elem()
		}
	}

	res := &collectionResolver{
		isSlice:   isSlice,
		reflValue: inValue,
		reflType:  inType,
	}

	if inType.Kind() != reflect.Struct {
		return res, errors.New("non-structs are not allowed as database input")
	}

	res.objName = inType.Name()
	if res.objName == "" {
		return res, errors.New("un-named types (nested type) are not allowed")
	}
	res.dbFolder = db.location(fmt.Sprintf("%x", sha256.New().Sum([]byte(res.objName))[:]))

	metaObjectRes, ok := inType.FieldByName("M")
	if !ok {
		return res, errors.New("No filedb.M in struct make sure to add it: struct{ filedb.M }")
	}
	metaObject := metaObjectRes.Type
	if metaObject.Kind() == reflect.Invalid {
		return res, errors.New("No filedb.M in struct make sure to add it: struct{ filedb.M }")
	}

	metaObjectTypeString := metaObject.String()
	if metaObjectTypeString != reflect.TypeOf(M{}).String() {
		return res, errors.New("Field M must be of type filedb.M not " + metaObjectTypeString)
	}

	if checkCollection {
		// Check if this struct is inside of collections
		found := false
		for key := range db.collections {
			found = key == res.objName
			if found {
				break
			}
		}
		if !found {
			return res, fmt.Errorf("%s not whitelisted, make sure to whitelist %s or check the whitelisting errors", res.objName, res.objName)
		}
	}

	return res, nil
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

// collectionInfo contains some meta data about the collection
type collectionInfo struct {
	IndexableKeys []string
}

// NewDB creates a new db
func NewDB(location string) (*DB, error) {
	db := &DB{
		rootDir:     location,
		collections: map[string]collection{},
	}

	err := ioutil.WriteFile(db.location(".initCheck"), []byte("This file is here to test premissions"), 0777)
	if err != nil {
		return nil, err
	}

	return db, nil
}
