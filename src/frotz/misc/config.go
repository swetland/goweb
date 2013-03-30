package misc

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"text/scanner"
)

type fieldMap map[string]reflect.Value
type sectionMap map[string]fieldMap

type Configuration struct {
	sections sectionMap
}

func (p *Configuration) AddSection(name string, ifc interface{}) {
	var v = reflect.ValueOf(ifc).Elem()
	var t = v.Type()

	var fields = make(fieldMap)

	for i := 0; i < t.NumField(); i++ {
		value := v.Field(i)
		if !value.CanSet() {
			continue;
		}
		if value.Kind() != reflect.String {
			continue;
		}
		fields[t.Field(i).Name] = value
	}

	if p.sections == nil {
		p.sections = make(sectionMap)
	}
	p.sections[name] = fields
}

type ParseError struct {
	why string
	pos scanner.Position
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.pos, e.why)
}

func prettyname(tok rune) string {
	switch tok {
	case scanner.EOF: return "End-of-File"
	case scanner.Ident: return "Identifier"
	case scanner.Int: return "Integer"
	case scanner.Float: return "Float"
	case scanner.Char: return "Char"
	case scanner.String: return "String"
	case scanner.RawString: return "RawString"
	case scanner.Comment: return "Comment"
	default: return fmt.Sprintf("'%c'", tok)
	}
}

func match(s *scanner.Scanner, expected rune) error {
	actual := s.Scan()
	if actual != expected {
		return ParseError{fmt.Sprintf("Expected %s, but found %s.",
			prettyname(expected), prettyname(actual)), s.Position}
	}
	return nil
}

func (p *Configuration) parseLine(s *scanner.Scanner, sname string, fields fieldMap) error {
	var err error
	name := s.TokenText()
	if err = match(s, '='); err != nil {
		return err;
	}
	if err = match(s, scanner.String); err != nil {
		return err
	}
	field, ok := fields[name]
	if !ok {
		return ParseError{
			fmt.Sprintf("Section '%s' has no item named '%s'.\n",
				sname, name), s.Position}
	}
	value, err := strconv.Unquote(s.TokenText())
	if err != nil {
		return ParseError{"Invalid String Constant",s.Position}
	}
	field.SetString(value)
	return nil
}

func (p *Configuration) parseSection(s *scanner.Scanner) error {
	name := s.TokenText()
	fields, ok := p.sections[name]
	if !ok {
		return ParseError{
			fmt.Sprintf("Invalid section '%s'.", name),
			s.Position }
	}
	if err := match(s, '{'); err != nil {
		return err
	}
	for {
		tok := s.Scan();
		if tok == '}' {
			return nil
		}
		if tok == scanner.Ident {
			if err := p.parseLine(s, name, fields); err != nil {
				return err
			}
			continue
		}
		return ParseError{
			fmt.Sprintf("Expected Identifier or '}', but found %s.",
				prettyname(tok)), s.Position}
	}
}

func (p *Configuration) parseConfig(s *scanner.Scanner) error {
	for {
		tok := s.Scan();

		if (tok == scanner.EOF) {
			return nil
		}

		if (tok == scanner.Ident) {
			if err := p.parseSection(s); err != nil {
				return err
			}
			continue;
		}

		return ParseError{
			fmt.Sprintf("Expected Identifier, but found %s.",
				prettyname(tok)), s.Position}
	}
}

func (p *Configuration) Parse(r io.Reader) error {
	var s scanner.Scanner
	s.Init(r)
	s.Mode = scanner.ScanIdents | scanner.ScanComments |
		scanner.SkipComments | scanner.ScanStrings
	return p.parseConfig(&s)
}


