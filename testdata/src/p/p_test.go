package p_test

import (
	"database/sql"
	"time"
)

func invalidCases() { //nolint:unused // ignore
	db, _ := sql.Open("fake", "user:password@/dbname")

	timeoutNoUnit := 1
	timeoutWithUnit := 1 * time.Second
	timeoutWithUnit2 := time.Second * 1

	db.SetConnMaxLifetime(1)                              // want `missing a valid time unit`
	db.SetConnMaxLifetime(1 * 60)                         // want `missing a valid time unit`
	db.SetConnMaxLifetime(1 * time.Duration(60))          // want `missing a valid time unit`
	db.SetConnMaxLifetime(time.Duration(timeoutNoUnit))   // want `missing a valid time unit`
	db.SetConnMaxLifetime(time.Duration(1))               // want `missing a valid time unit`
	db.SetConnMaxLifetime(1 * time.Millisecond)           // want `missing a valid time unit`
	db.SetConnMaxLifetime(returnTimeDuration())           // want `potentially missing a time unit`
	db.SetConnMaxLifetime(+1)                             // want `operator is not -`
	db.SetConnMaxLifetime(1 * time.Second)                // want `time is less than minimum required`
	db.SetConnMaxLifetime(time.Second * 1)                // want `time is less than minimum required`
	db.SetConnMaxLifetime(time.Duration(1) * time.Second) // want `time is less than minimum required`
	db.SetConnMaxLifetime(time.Second * time.Duration(1)) // want `time is less than minimum required`
	db.SetConnMaxLifetime(timeoutWithUnit)                // want `time is less than minimum required`
	db.SetConnMaxLifetime(timeoutWithUnit2)               // want `time is less than minimum required`
	db.SetConnMaxLifetime(time.Second)                    // want `time is less than minimum required`
}

func validCases() { //nolint:unused // ignore
	db, _ := sql.Open("fake", "user:password@/dbname")

	timeoutWithUnit := 60 * time.Second
	timeoutWithUnit2 := time.Second * 60

	db.SetConnMaxLifetime(0)
	db.SetConnMaxLifetime(-1)
	db.SetConnMaxLifetime(60 * time.Second)
	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxLifetime(time.Second * 60)
	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetConnMaxLifetime(time.Hour * 1)
	db.SetConnMaxLifetime(time.Duration(60) * time.Second)
	db.SetConnMaxLifetime(time.Duration(1) * time.Minute)
	db.SetConnMaxLifetime(time.Duration(1) * time.Hour)
	db.SetConnMaxLifetime(time.Second * time.Duration(60))
	db.SetConnMaxLifetime(time.Minute * time.Duration(1))
	db.SetConnMaxLifetime(time.Hour * time.Duration(1))
	db.SetConnMaxLifetime(timeoutWithUnit)
	db.SetConnMaxLifetime(timeoutWithUnit2)
	db.SetConnMaxLifetime(time.Minute)
}

func returnTimeDuration() time.Duration { //nolint:unused // ignore
	return 1
}
