# conncheck - Go linter for db configuration

![Latest release](https://img.shields.io/github/v/release/dozyio/conncheck)
[![CI](https://github.com/dozyio/conncheck/actions/workflows/release.yml/badge.svg)](https://github.com/dozyio/conncheck/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dozyio/conncheck)](https://goreportcard.com/report/github.com/dozyio/conncheck)
[![Coverage Status](https://coveralls.io/repos/github/dozyio/conncheck/badge.svg?branch=main)](https://coveralls.io/github/dozyio/conncheck?branch=main)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)

## Installing

`go install github.com/dozyio/conncheck/cmd/conncheck@latest`

## Running

`conncheck ./...`

### Options

`-minsec` The minimum seconds for SetConnMaxLifetime (default 60). This is a
best effort static check but can be unreliable as the value is often set at
runtime.

`-packages` A comma-separated list of packages to trigger linting (default database/sql,gorm.io/gorm,github.com/jmoiron/sqlx)

`-timeunits` A comma-separated list of time units to validate against (default Second,Minute,Hour)

`-printast` Print the AST, useful for debugging

## Linting

Currently the linter only checks [`db.SetConnMaxLifetime`](https://pkg.go.dev/database/sql#DB.SetConnMaxLifetime).

### db.SetConnMaxLifetime()

Checks that [`db.SetConnMaxLifetime`](https://pkg.go.dev/database/sql#DB.SetConnMaxLifetime)
is set to a reasonable value to optimise performance. `SetConnMaxLifetime`
accepts a `time.Duration` that is in nanoseconds but is often configured
incorrectly. This can lead to performance issues, such as a new connection on
every request.

## Recommendations

* When reading the value for `SetConnMaxLifetime` from a configuration file, use
`time.ParseDuration()` to ensure a time unit is set.
