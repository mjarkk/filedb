# `FileDB` A dumb but usefull database

Have you hever thought *"Wait a second i'm using a key value store as nosql database because i want to store flexable document but have to meany of them to store them in one json file"*?  
Well then you where probebly searching for this, this database aims to solve that problem by providing an easy to use nosql database that is server less.  

NOTE: This is probebly not a stable database and thus you should not use this in a production envourment where the data you are storing is important.  

## Feathers
- Easy to use
- Small with less than 1000 lines of go code and 1 dependnecy [go.uuid](github.com/satori/go.uuid)

## Limitations
- You cannot make OR queries nor make complicated number queries like greather than, etc...
- The db is currently really dependent on the file system so if the fs is shitty or the hardware this db wont work well
- Every object is stored as json using go's json package so the json field tag will be used what might cause some problems

## Get started
```go
func main() {
	db, err := filedb.NewDB("db")
	handleErr(err)

	type User struct{
		fileDB.M
		Username string `json:"Username"`
		Password string `json:"Password"`
	}

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
```

## What are these KV things?
TL;DR Short for **k**ey and **v**alue, they are simply a pair of a key and value used for searching ment to be set global and used by everything.  

Since this database is limited to the naming of the created files we cannot just search for everything inside a struct.  
Because of that i want these things pre defined and a KV (**k**ey **v**alue) object seems to be the best option for this.  
With a KV you can pre define the key and later set the value.  
The creation of a KV is ment to be dumb long because then a api user will refactor them away globally somewhat automaticly what hopefully results in less user error in the long run.  

## Compaired to other types of databases

| * | FileDB | SqlLite | Most NoSql DBs | Key-value stores |
| - | - | - | - | - |
| **Easy to use** | Yes | No *(Setting up and using requires quite a bit of nolange and time)* | Yes | Yes |
| **Can query more than a key** | Yes | Yes | Yes | No |
| **Can make advanced queries** | No | Yes | Yes | No |
| **Serverless & C-less** (no imported c code) | Yes | Yes | No *(Most projects imports c code and thus adding complexity to a program)* | Yes |
| **Code size** | Small | Medium | Huge | Huge |

## But why this over something like gorm combined with sqlite
Mostly personal preference, gorm is an amazing library but has some major limitations that remove the fun working with it for me at least.  
- You cannot use maps *(In this library you can)*
- Every nested struct is a database table and thus every nested struct needs an ID with as result you need to reference the orignal struct in the nested one, WHAAAA this is soo dumb *(This library doesn't create a new collection for every struct)*
- The oldschool sql relations makes it so you need to inform gorm about how tables are called when joined *(Here you don't have to because there are no relations)*
- Your code needs to match sql's way of thinking, sounds dumb but you can't just start creating structs and gorm will figure out the relations as long as there is a gorm.m, You can arguee that this is for the good and i agree if you are running a sql database but that's exacly my problem i don't want a sql database because of these problems.
