package compute

import (
	"context"
	"regexp"
	"testing"

	"github.com/hexops/autogold"

	"github.com/sourcegraph/sourcegraph/internal/search/result"
)

func Test_output(t *testing.T) {
	test := func(input string, cmd *Output) string {
		result, err := output(context.Background(), input, cmd.MatchPattern, cmd.OutputPattern, cmd.Separator)
		if err != nil {
			return err.Error()
		}
		return result
	}

	autogold.Want(
		"regexp search outputs only digits",
		"(1)~(2)~(3)~").
		Equal(t, test("a 1 b 2 c 3", &Output{
			MatchPattern:  &Regexp{Value: regexp.MustCompile(`(\d)`)},
			OutputPattern: "($1)",
			Separator:     "~",
		}))
}

func TestRun(t *testing.T) {
	test := func(input string, fm *result.FileMatch) string {
		q, _ := Parse(input)
		res, err := q.Command.Run(context.Background(), fm)
		if err != nil {
			return err.Error()
		}
		return res.(*Text).Value
	}

	autogold.Want(
		"template substitution",
		"(1)\n(2)\n(3)\n").
		Equal(t, test(`content:output((\d) -> $Path:($1))`, &result.FileMatch{
			File: result.File{Path: "my/awesome/path"},
			LineMatches: []*result.LineMatch{
				{
					Preview: "a 1 b 2 c 3",
				},
			},
		}))
}
