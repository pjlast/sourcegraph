package compute

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
)

type Output struct {
	MatchPattern  MatchPattern
	OutputPattern string
	Separator     string
}

func (c *Output) String() string {
	return fmt.Sprintf("Output with separator: (%s) -> (%s) separator: %s", c.MatchPattern.String(), c.OutputPattern, c.Separator)
}

func extract(str string) (name string, num int, rest string, ok bool) {
	if len(str) < 2 || str[0] != '$' {
		return
	}
	str = str[1:]
	i := 0
	for i < len(str) {
		rune, size := utf8.DecodeRuneInString(str[i:])
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) && rune != '_' {
			break
		}
		i += size
	}
	if i == 0 {
		// empty name is not okay
		return
	}
	name = str[:i]

	// Parse number.
	num = 0
	for i := 0; i < len(name); i++ {
		if name[i] < '0' || '9' < name[i] || num >= 1e8 {
			num = -1
			break
		}
		num = num*10 + int(name[i]) - '0'
	}
	// Disallow leading zeros.
	if name[0] == '0' && len(name) > 1 {
		num = -1
	}

	rest = str[i:]
	ok = true
	return
}

func expand(re *regexp.Regexp, dst []byte, template string, bsrc []byte, src string, match []int) []byte {
	for len(template) > 0 {
		i := strings.Index(template, "$")
		if i < 0 {
			break
		}
		dst = append(dst, template[:i]...)
		template = template[i:]
		name, num, rest, ok := extract(template)
		log15.Info("seen", "", name)
		if !ok {
			log15.Info("seen", "not ok", name)
			// Malformed; treat $ as raw text.
			dst = append(dst, '$')
			template = template[1:]
			continue
		}
		template = rest
		if num >= 0 {
			if 2*num+1 < len(match) && match[2*num] >= 0 {
				if bsrc != nil {
					dst = append(dst, bsrc[match[2*num]:match[2*num+1]]...)
				} else {
					dst = append(dst, src[match[2*num]:match[2*num+1]]...)
				}
			}
		} else {
			for i, namei := range re.SubexpNames() {
				if name == namei && 2*i+1 < len(match) && match[2*i] >= 0 {
					if len(src[match[2*i]:match[2*i+1]]) > 0 {
						dst = append(dst, src[match[2*i]:match[2*i+1]]...)
						break
					} else {
						dst = append(dst, '$')
						dst = append(dst, name...)
						break
					}
				}
			}
		}
	}
	dst = append(dst, template...)
	return dst
}

func substituteRegexp(content string, match *regexp.Regexp, replacePattern, separator string) string {
	var b strings.Builder
	for _, submatches := range match.FindAllStringSubmatchIndex(content, -1) {
		b.Write(expand(match, []byte{}, replacePattern, nil, content, submatches))
		b.WriteString(separator)
	}
	return b.String()
}

func output(ctx context.Context, fragment string, matchPattern MatchPattern, replacePattern string, separator string) (string, error) {
	var newFragment string
	switch match := matchPattern.(type) {
	case *Regexp:
		newFragment = substituteRegexp(fragment, match.Value, replacePattern, separator)
	case *Comby:
		return "", errors.New("unsupported")
	}
	return newFragment, nil
}

func (c *Output) Run(ctx context.Context, fm *result.FileMatch) (Result, error) {
	lines := make([]string, 0, len(fm.LineMatches))
	for _, line := range fm.LineMatches {
		lines = append(lines, line.Preview)
	}
	fragment := strings.Join(lines, "\n")
	newFragment, err := output(ctx, fragment, c.MatchPattern, c.OutputPattern, c.Separator)
	if err != nil {
		return nil, err
	}
	log15.Info("new fragment", newFragment)
	template, err := templatize(newFragment)
	if err != nil {
		return nil, err
	}
	var result bytes.Buffer
	if err := template.Execute(&result, fm); err != nil {
		return nil, err
	}
	return &Text{Value: result.String(), Kind: "output"}, nil
}
