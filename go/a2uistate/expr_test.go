package a2uistate

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	path := func(p string) map[string]any { return map[string]any{"path": p} }
	call := func(name string, args map[string]any) map[string]any {
		return map[string]any{"call": name, "args": args}
	}
	tests := []struct {
		in   string
		want []any
	}{
		{"", nil},
		{"plain", []any{"plain"}},
		{"Hi ${/name}!", []any{"Hi ", path("/name"), "!"}},
		{"${name}${ /a/b }", []any{path("name"), path("/a/b")}},
		{`\${x} ${x}`, []any{"${", "x} ", path("x")}},
		{`a\b`, []any{`a\b`}},
		{"${'lit'}${\"q\\\"t\"}${42}${-1.5e2}${true}${false}${null}", []any{"lit", `q"t`, 42.0, -150.0, true, false}},
		{"${'a\\nb'}", []any{"a\nb"}},
		{"${${/x}}", []any{path("/x")}},
		{"${}", nil},
		{"${now()}", []any{call("now", map[string]any{})}},
		{"${@index(offset: 1)}", []any{call("@index", map[string]any{"offset": 1.0})}},
		{
			"${formatDate(value:${/d}, format:'MM-dd')}",
			[]any{call("formatDate", map[string]any{"value": path("/d"), "format": "MM-dd"})},
		},
		{"${upper(v: ${now()})}", []any{call("upper", map[string]any{"v": call("now", map[string]any{})})}},
		{"${f(s: '}')}", []any{call("f", map[string]any{"s": "}"})}},
		{"${items.0}", []any{path("items.0")}},
	}
	for _, tt := range tests {
		got, err := parseTemplate(tt.in, 0)
		if err != nil {
			t.Errorf("parseTemplate(%q): %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseTemplate(%q) = %#v, want %#v", tt.in, got, tt.want)
		}
	}
}

func TestParseTemplateErrors(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"${/x", "unclosed interpolation"},
		{"${a b}", "unexpected characters"},
		{"${f(x 1)}", "expected ':'"},
		{"${f(x: 1}", "expected ')'"},
		{"${1.2.3}", "invalid number literal"},
		{"${@x}", "unexpected characters"},
		{strings.Repeat("${", 200) + strings.Repeat("}", 200), "nested too deeply"},
		{strings.Repeat("x", maxTemplateLen+1), "exceeds"},
		{strings.Repeat("${a}", maxTemplateParts+1), "more than"},
	}
	for _, tt := range tests {
		_, err := parseTemplate(tt.in, 0)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseTemplate(%.20q) error = %v, want %q", tt.in, err, tt.want)
		}
	}
}
