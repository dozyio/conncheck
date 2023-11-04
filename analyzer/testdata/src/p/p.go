package p

import (
	"database/sql"
	"time"
)

func invalidCases() { //nolint:unused // ignore
	db, _ := sql.Open("fake", "user:password@/dbname") //nolint:errcheck // ignore

	timeout := 1

	db.SetConnMaxLifetime(1)                      // want `include a time unit e.g. time.Second`
	db.SetConnMaxLifetime(1 * 60)                 // want `include a time unit e.g. time.Second`
	db.SetConnMaxLifetime(1 * time.Duration(60))  // want `include a time unit e.g. time.Second`
	db.SetConnMaxLifetime(time.Duration(timeout)) // want `include a time unit e.g. time.Second`
	db.SetConnMaxLifetime(time.Duration(1))       // want `include a time unit e.g. time.Second`
	db.SetConnMaxLifetime(+1)                     // want `operator is not -`
}

func validCases() { //nolint:unused // ignore
	db, _ := sql.Open("fake", "user:password@/dbname") //nolint:errcheck // ignore

	timeout := 1 * time.Second

	db.SetConnMaxLifetime(0)
	db.SetConnMaxLifetime(-1)
	db.SetConnMaxLifetime(1 * time.Second)
	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxLifetime(time.Second * 1)
	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetConnMaxLifetime(time.Hour * 1)
	db.SetConnMaxLifetime(1 * 60 * time.Minute)
	db.SetConnMaxLifetime(time.Duration(1) * time.Second)
	db.SetConnMaxLifetime(time.Duration(1) * time.Minute)
	db.SetConnMaxLifetime(time.Duration(1) * time.Hour)
	db.SetConnMaxLifetime(time.Second * time.Duration(1))
	db.SetConnMaxLifetime(time.Minute * time.Duration(1))
	db.SetConnMaxLifetime(time.Hour * time.Duration(1))
	db.SetConnMaxLifetime(timeout)
	db.SetConnMaxLifetime(time.Minute)
	db.SetConnMaxLifetime(GetMaxLifetimeOk())
}

func GetMaxLifetimeOk() time.Duration {
	return 1 * time.Second
}

func GetMaxLifetimeBad() time.Duration {
	return 1
}
