# conncheck - Go linter for db configuration

![Latest release](https://img.shields.io/github/v/release/dozyio/conncheck)
[![CI](https://github.com/dozyio/conncheck/actions/workflows/release.yml/badge.svg)](https://github.com/dozyio/conncheck/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dozyio/conncheck)](https://goreportcard.com/report/github.com/dozyio/conncheck?dummy=unused)
[![Coverage Status](https://coveralls.io/repos/github/dozyio/conncheck/badge.svg?branch=main)](https://coveralls.io/github/dozyio/conncheck?branch=main)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)

## db.SetConnMaxLifetime

Checks that [`db.SetConnMaxLifetime`](https://pkg.go.dev/database/sql#DB.SetConnMaxLifetime)
is set to a reasonable value to optimise performance. It accepts a
`time.Duration` that is in nanoseconds but is often configured incorrectly,
leading to performance issues, such as a new connection on every request.
