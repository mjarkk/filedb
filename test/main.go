package main

import (
	"fmt"
	"log"

	"github.com/mjarkk/filedb"
)

// User is a db user
type User struct {
	filedb.M
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db, err := filedb.NewDB("db")
	handleErr(err)

	// Set the search keys
	userKV := filedb.NewKV("Username")
	passKV := filedb.NewKV("Password")

	// Add database
	err = db.Whitelist(
		&User{},
		passKV,
		userKV,
	)
	handleErr(err)

	// Insert a user
	newUser := &User{
		Username: "root",
		Password: "my-secret-password",
	}
	err = db.Save(newUser)
	handleErr(err)
	fmt.Println("User ID:", newUser.ID)

	// Find multiple documents
	users := []User{}
	err = db.Find(&users)
	handleErr(err)
	fmt.Println("First user:", users[0].Username, ",Total Users:", len(users))

	// Find single document, with a filter
	user := &User{}
	err = db.Find(user, filedb.IDkv.V(newUser.ID))
	handleErr(err)

	fmt.Println("Found user:", user.Username)

	// Remove everything
	// You can also add fillin indexable data in the user and if so that will be used as filter
	// Like: &User{Username: "root"}
	err = db.Delete(&User{})
	handleErr(err)
}
