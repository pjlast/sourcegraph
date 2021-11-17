package compute

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"text/template"

	"github.com/inconshreveable/log15"
)

type GlobalAttr string

const (
	Repository = "repository"
	Path       = "path"
	Commit     = "commit"
)

type Atom interface {
	atom()
	String() string
}

type Variable struct {
	Value     string
	Attribute string
}

type Constant string

func (Variable) atom() {}
func (Constant) atom() {}

func (v Variable) String() string {
	if v.Attribute != "" {
		return v.Value + "." + v.Attribute
	}
	return v.Value
}
func (c Constant) String() string { return string(c) }

type Template []Atom

const varAllowed = "abcdefghijklmnopqrstuvwxyzABCEDEFGHIJKLMNOPQRSTUVWXYZ1234567890_."

func parseTemplate(buf []byte) (*Template, error) {
	// Tracks whether the current token is a variable.
	var isVariable bool

	var start int
	var r rune
	var token []rune
	var result []Atom

	next := func() rune {
		r, start := utf8.DecodeRune(buf)
		buf = buf[start:]
		return r
	}

	appendAtom := func(atom Atom) {
		if a, ok := atom.(Constant); ok && len(a) == 0 {
			return
		}
		if a, ok := atom.(Variable); ok && len(a.Value) == 0 {
			return
		}
		result = append(result, atom)
		// Reset token, but reuse the backing memory
		token = token[:0]
	}

	for len(buf) > 0 {
		r = next()
		switch r {
		case '$':
			if len(buf[start:]) > 0 {
				if r, _ = utf8.DecodeRune(buf); strings.ContainsRune(varAllowed, r) {
					// Start of a recognized variable
					if isVariable {
						appendAtom(Variable{Value: string(token)}) // Push variable
						isVariable = false
					} else {
						appendAtom(Constant(token))
					}
					token = append(token, '$')
					isVariable = true
					continue
				}
				// Something else, push the '$' we saw and continue.
				token = append(token, '$')
				isVariable = false
				continue
			}
			// Trailing '$'
			if isVariable {
				appendAtom(Variable{Value: string(token)}) // Push variable
				isVariable = false
			} else {
				appendAtom(Constant(token))
			}
			token = append(token, '$')
		case '\\':
			isVariable = false
			if len(buf[start:]) > 0 {
				r = next()
				switch r {
				case 'n':
					token = append(token, '\n')
				case 'r':
					token = append(token, '\r')
				case 't':
					token = append(token, '\t')
				case '\\', '$', ' ', '.':
					token = append(token, r)
				default:
					token = append(token, '\\', r)
				}
				continue
			}
			// Trailing '\'
			token = append(token, '\\')
		default:
			if isVariable && !strings.ContainsRune(varAllowed, r) {
				appendAtom(Variable{Value: string(token)}) // Push variable
				isVariable = false
			}
			token = append(token, r)
		}
	}
	if len(token) > 0 {
		if isVariable {
			appendAtom(Variable{Value: string(token)})
		} else {
			appendAtom(Constant(token))
		}
	}
	t := Template(result)
	return &t, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func templatize(pattern string, exclude []string) (string, error) {
	exclude = append(exclude, "1", "2", "3", "4", "5", "6", "7") // XXX hardcode
	t, err := parseTemplate([]byte(pattern))
	if err != nil {
		return "", err
	}
	var templatized []string
	for _, atom := range *t {
		switch a := atom.(type) {
		case Constant:
			templatized = append(templatized, string(a))
		case Variable:
			if contains(exclude, a.Value[1:]) {
				// Leave alone excluded variables (corresponding to regex capture groups).
				templatized = append(templatized, a.Value[:])
				continue
			}
			templateVar := strings.Title(a.Value[1:])
			templatized = append(templatized, `{{.`+templateVar+`}}`)
		}
	}
	return strings.Join(templatized, ""), nil
}

type MetaValue struct {
	Repo    string
	Path    string
	Content string
}

func substituteMetaVariables(pattern string, exclude []string, value interface{}) (string, error) {
	templated, err := templatize(pattern, exclude)
	log15.Info("templated", "t", templated)
	if err != nil {
		return "", err
	}
	t, err := template.New("dontcare").Parse(templated)
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	if err := t.Execute(&result, value); err != nil {
		return "", err
	}
	return result.String(), nil
}

func substituteTemplateMetaVariables(matchPattern MatchPattern, replacePattern string, value interface{}) (string, error) {
	var result string
	var err error
	switch match := matchPattern.(type) {
	case *Regexp:
		result, err = substituteMetaVariables(replacePattern, match.Value.SubexpNames(), value)
	case *Comby:
		result, err = substituteMetaVariables(replacePattern, []string{}, value)
	}
	return result, err
}

func toJSON(atom Atom) interface{} {
	switch a := atom.(type) {
	case Constant:
		return struct {
			Value string `json:"constant"`
		}{
			Value: string(a),
		}
	case Variable:
		return struct {
			Value     string `json:"variable"`
			Attribute string `json:"attribute,omitempty"`
		}{
			Value:     a.Value,
			Attribute: a.Attribute,
		}
	}
	panic("unreachable")
}

func toJSONString(template *Template) string {
	var jsons []interface{}
	for _, atom := range *template {
		jsons = append(jsons, toJSON(atom))
	}
	json, _ := json.Marshal(jsons)
	return string(json)
}
