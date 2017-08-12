package main

import (
	"io/ioutil"
	"testing"
	"os"
	"time"
)

//
//  Temporary location for database
//
var path string

//
// Create a temporary database
//
func FakeDB(){
	p, err := ioutil.TempDir(os.TempDir(), "prefix")
	if ( err == nil ) {
		path = p
	}

	//
	// Setup the tables.
	//
	SetupDB( p + "/db.sql")

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	//
	// Add some records
	stmt, err := tx.Prepare("INSERT INTO reports(yaml_file,executed_at) values(?,?)" )
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	count := 0

	for count < 10  {
		now  := time.Now().Unix()
		days := int64(60 * 60 * 24 * count)

		now -= days
		stmt.Exec("", now)
		count += 1
	}
	tx.Commit()
}

func TestPrune(t *testing.T) {

	// Create a fake database
	FakeDB();


	//
	// Count records and assume we have some.
	//
	old,_ := countReports()

	if ( old != 10 ) {
		t.Errorf("We have %d reports, not 10", old )
	}

	//
	// Run the prune
	//
	pruneReports(5,false)

	//
	// Count them again
	//
	new,_ := countReports()

	if ( new != 6 ) {
		t.Errorf("We have %d reports, not 5", new )
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}
