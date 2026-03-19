package slogrus

import "fmt"

func sprint(args []interface{}) string         { return fmt.Sprint(args...) }
func sprintf(f string, a []interface{}) string { return fmt.Sprintf(f, a...) }
func sprintln(args []interface{}) string       { return fmt.Sprintln(args...) }
