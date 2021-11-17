package compute

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/comby"
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

func substituteRegexp(content string, match *regexp.Regexp, replacePattern, separator string) string {
	var b strings.Builder
	for _, submatches := range match.FindAllStringSubmatchIndex(content, -1) {
		b.Write(match.ExpandString([]byte{}, replacePattern, content, submatches))
		b.WriteString(separator)
	}
	return b.String()
}

func output(ctx context.Context, fragment string, matchPattern MatchPattern, replacePattern string, separator string) (*Text, error) {
	var newContent string
	var err error
	switch match := matchPattern.(type) {
	case *Regexp:
		newContent = substituteRegexp(fragment, match.Value, replacePattern, separator)
	case *Comby:
		newContent, err = comby.Outputs(ctx, comby.Args{
			Input:           comby.FileContent(fragment),
			MatchTemplate:   match.Value,
			RewriteTemplate: replacePattern,
			Matcher:         ".generic", // TODO(rvantoner): use language or file filter
			ResultKind:      comby.NewlineSeparatedOutput,
			NumWorkers:      0,
		})
		if err != nil {
			return nil, err
		}

	}
	return &Text{Value: newContent, Kind: "output"}, nil
}

func (c *Output) Run(ctx context.Context, fm *result.FileMatch) (Result, error) {
	lines := make([]string, 0, len(fm.LineMatches))
	for _, line := range fm.LineMatches {
		lines = append(lines, line.Preview)
	}
	fragment := strings.Join(lines, "\n")
	substitutedOutputPattern := c.OutputPattern
	substitutedOutputPattern, err := substituteTemplateMetaVariables(
		c.MatchPattern,
		c.OutputPattern,
		&MetaValue{
			Repo:    string(fm.Repo.Name),
			Path:    fm.Path,
			Content: fragment, // FIXME: should be just the matched part once we use files ($0 for regexp)
		})
	if err != nil {
		return nil, err
	}
	log15.Info("substituted", "x", substitutedOutputPattern)
	return output(ctx, fragment, c.MatchPattern, substitutedOutputPattern, c.Separator)
}
