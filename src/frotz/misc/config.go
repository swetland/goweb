// Copyright 2013 Brian Swetland <swetland@frotz.net>

// Package misc provides some small utility items that don't yet
// have a better home.
//
package misc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
)

type fieldMap map[string]reflect.Value
type sectionMap map[string]fieldMap

// Configuration provides a mechanism for a number of structs (sections)
// to be initialized from a simple config file.
type Configuration struct {
	sections sectionMap
}

// AddSection creates a new named section in the configuration file
// which contains allowed fields named using the struct field tags
// of the writeable string fields in the struct provided via ifc.
//
// The provided struct is not only the template for valid config
// file syntax, but also will be filled in by the values in the
// config file when Parse() is called
//
//  type Size struct {
//      Width string `width`
//      Height string `height`
//  }
//  cfg.AddSection("size", &Size)
//
func (cfg *Configuration) AddSection(name string, ifc interface{}) {
	var v = reflect.ValueOf(ifc).Elem()
	var t = v.Type()

	var fields = make(fieldMap)

	for i := 0; i < t.NumField(); i++ {
		value := v.Field(i)
		tag := string(t.Field(i).Tag)
		if len(tag) == 0 {
			continue
		}
		if !value.CanSet() {
			continue
		}
		if value.Kind() != reflect.String {
			continue
		}
		fields[tag] = value
	}

	if cfg.sections == nil {
		cfg.sections = make(sectionMap)
	}
	cfg.sections[name] = fields
}

type ParseError struct {
	What string
	Line int
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%d: %s", e.Line, e.What)
}

var sepEQ = []byte{'='}

// Parse attempts to parse a configuration file using the registered
// sections and fields.  Referencing nonexistant sections or fields
// is an error
//
//  # comments like this
//  [size]
//  width = 17
//  height = 42
//
// Blank lines are ignored.  Whitespace surrounding the = and at the
// start or end of lines is also ignored.
//
func (cfg *Configuration) Parse(r io.Reader) error {
	var sname string
	var fields fieldMap = nil
	var ok bool
	lineno := 0
	s := bufio.NewScanner(r)
	for s.Scan() {
		lineno++
		line := s.Bytes()
		line = bytes.TrimSpace(line)
		n := len(line)

		// ignore blank lines or comments
		if n == 0 || line[0] == '#' {
			continue
		}

		// [section] makes a new section active
		if line[0] == '[' && line[n-1] == ']' {
			sname = string(bytes.TrimSpace(line[1 : n-1]))
			fields, ok = cfg.sections[sname]
			if !ok {
				return &ParseError{fmt.Sprintf(
					"Unknown config section [%s]",
					sname), lineno}
			}
			continue
		}

		// other lines must be foo=bar style
		parts := bytes.Split(line, sepEQ)
		if len(parts) != 2 {
			return &ParseError{"Invalid Assignment", lineno}
		}
		if fields == nil {
			return &ParseError{"No section specified", lineno}
		}
		name := string(bytes.TrimSpace(parts[0]))
		value := string(bytes.TrimSpace(parts[1]))
		v, ok := fields[name]
		if !ok {
			return &ParseError{fmt.Sprintf(
				"Section [%s] has no variable '%s'",
				sname, name), lineno}
		}
		v.SetString(value)
	}

	// succeeded... unless there was an io error
	return s.Err()
}
