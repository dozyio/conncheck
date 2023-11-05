package conncheck

import "strings"

type stringSliceValue struct {
	slice []string
}

// Implement the Set method of the flag.Value interface
func (ss *stringSliceValue) Set(s string) error {
	ss.slice = strings.Split(s, ",")
	return nil
}

// Implement the String method of the flag.Value interface
func (ss *stringSliceValue) String() string {
	return strings.Join(ss.slice, ",")
}

// Utility function to append a slice of strings to the value
func (ss *stringSliceValue) SetSlice(s []string) {
	ss.slice = append(ss.slice, s...)
}
