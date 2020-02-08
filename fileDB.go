package filedb

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"time"

	uuid "github.com/satori/go.uuid"
)

// M needs to be inside a struct for it to be a valid
type M struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DB contains the database information
type DB struct {
	rootDir     string
	collections map[string]collection // a map of all allowed struct in the database
}

type collection struct {
	indexKeys []string
	folder    string
}

// location returns the full location of a file
func (db *DB) location(add ...string) string {
	return path.Join(append([]string{db.rootDir}, add...)...)
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
