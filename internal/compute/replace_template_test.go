package compute

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hexops/autogold"
)

func Test_parse(t *testing.T) {
	test := func(input string) string {
		t, err := parseReplaceTemplate([]byte(input))
		if err != nil {
			return fmt.Sprintf("Error: %s", err)
		}
		return toJSONString(t)
	}

	autogold.Want(
		"basic template",
		`[{"constant":"artifcats: "},{"variable":"$repo"}]`).
		Equal(t, test("artifcats: $repo"))

	autogold.Want(
		"multiple $",
		`[{"constant":"$"},{"variable":"$foo"},{"constant":" $"},{"variable":"$bar"}]`).
		Equal(t, test("$$foo $$bar"))

	autogold.Want(
		"terminating variable",
		`[{"variable":"$repo"},{"constant":"(derp)"}]`).
		Equal(t, test(`$repo(derp)`))

	autogold.Want(
		"consecutive variables with separator",
		`[{"variable":"$repo"},{"constant":":"},{"variable":"$file"},{"constant":" "},{"variable":"$content"}]`).
		Equal(t, test(`$repo:$file $content`))

	autogold.Want(
		"consecutive variables no separator",
		`[{"variable":"$repo"},{"variable":"$file"}]`).
		Equal(t, test("$repo$file"))

	autogold.Want(
		"terminating variables with trailing $",
		`[{"constant":"$"},{"variable":"$foo"},{"variable":"$bar"},{"constant":"$"}]`).
		Equal(t, test("$$foo$bar$"))

	autogold.Want(
		"end-of-template variable",
		`[{"variable":"$bar"}]`).
		Equal(t, test("$bar"))

	autogold.Want(
		"space escaping",
		`[{"constant":"$repo "}]`).
		Equal(t, test(`$repo\ `))

	autogold.Want(
		"metachar escaping",
		`[{"constant":"$repo "}]`).
		Equal(t, test(`\$repo `))
}

func Test_templatize(t *testing.T) {
	test := func(input string, data interface{}) string {
		t, err := templatize(input)
		if err != nil {
			return fmt.Sprintf("Error: %s", err)
		}
		var result bytes.Buffer
		if err := t.Execute(&result, data); err != nil {
			return fmt.Sprintf("Error: %s", err)
		}
		return result.String()
	}

	autogold.Want(
		"substitute",
		"yo I'm substituted").
		Equal(t, test(`$repo`, struct{ Repo string }{Repo: "yo I'm substituted"}))
}
