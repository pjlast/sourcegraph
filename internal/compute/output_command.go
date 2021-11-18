package compute

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/comby"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
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

func (c *Output) Run(ctx context.Context, r result.Match) (Result, error) {
	switch m := r.(type) {
	case *result.FileMatch:
		content, err := git.ReadFile(ctx, m.Repo.Name, m.CommitID, m.Path, 0)
		if err != nil {
			return nil, err
		}
		outputPattern, err := substituteMetaVariables(
			c.OutputPattern,
			&MetaValue{
				Repo:    string(m.Repo.Name),
				Path:    m.Path,
				Commit:  string(m.CommitID),
				Content: string(content),
			},
		)
		if err != nil {
			return nil, err
		}
		return output(ctx, string(content), c.MatchPattern, outputPattern, c.Separator)

	case *result.CommitMatch:
		var content string
		if m.DiffPreview != nil {
			// This is a diff result. Use Body, which is actually
			// ```diff <...>``` markdown, because I don't think we
			// expose it without markdown.
			content = m.Body.Value
		} else {
			content = string(m.Commit.Message)
		}
		outputPattern, err := substituteMetaVariables(
			c.OutputPattern,
			&MetaValue{
				Repo:    string(m.Repo.Name),
				Commit:  string(m.Commit.ID),
				Author:  m.Commit.Author.Name,
				Date:    m.Commit.Committer.Date.Format("2006-01-02 15:04:05"),
				Email:   m.Commit.Committer.Email,
				Content: content,
			},
		)
		if err != nil {
			return nil, err
		}
		return output(ctx, content, c.MatchPattern, outputPattern, c.Separator)
	}
	return nil, nil
}
