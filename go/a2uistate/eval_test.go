package a2uistate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

func TestEvaluatorTypedArgs(t *testing.T) {
	e := &Evaluator{Data: evalModel(t)}
	tests := []struct {
		name string
		d    a2ui.DynamicBoolean
		want bool
	}{
		{"not", a2ui.Not(a2ui.BoolBinding("/on")), false},
		{"and", a2ui.And([]a2ui.DynamicBoolean{a2ui.BoolLiteral(true), a2ui.BoolBinding("/on")}), true},
		{"or", a2ui.Or([]a2ui.DynamicBoolean{a2ui.BoolBinding("/off"), a2ui.Not(a2ui.BoolBinding("/off"))}), true},
		{"validation", a2ui.And([]a2ui.DynamicBoolean{a2ui.BoolLiteral(true), a2ui.BoolFunc(*a2ui.Email(a2ui.StringBinding("/bad")).FunctionCall)}), false},
	}
	for _, tt := range tests {
		got, err := e.ResolveBoolean(tt.d, "")
		if err != nil || got != tt.want {
			t.Errorf("%s: ResolveBoolean = %v, %v, want %v", tt.name, got, err, tt.want)
		}
	}

	n, err := e.ResolveNumber(a2ui.Index(1), "/items/1")
	if n != 2 || err != nil {
		t.Errorf("ResolveNumber(Index(1)) = %v, %v, want 2", n, err)
	}
	s, err := e.ResolveString(a2ui.FormatString(a2ui.StringLiteral("${name}: ${count}")), "/items/0")
	if s != "Bob: 1" || err != nil {
		t.Errorf("ResolveString(FormatString) = %q, %v, want %q", s, err, "Bob: 1")
	}
	v, err := e.ResolveValue(a2ui.ValueFunc(*a2ui.Required(a2ui.ValueBinding("/name")).FunctionCall), "")
	if want := map[string]any{"valid": true}; !reflect.DeepEqual(v, want) || err != nil {
		t.Errorf("ResolveValue(Required) = %v, %v, want %v", v, err, want)
	}
}

func TestEvaluatorErrors(t *testing.T) {
	e := &Evaluator{Data: evalModel(t)}
	check := func(name string, err, want error) {
		t.Helper()
		if !errors.Is(err, want) {
			t.Errorf("%s: error = %v, want %v", name, err, want)
		}
		if err != nil && !strings.HasPrefix(err.Error(), "a2uistate: ") {
			t.Errorf("%s: error %q lacks the a2uistate: prefix", name, err)
		}
	}
	_, err := e.ResolveString(a2ui.DynamicString{}, "")
	check("unset string", err, ErrNoValue)
	_, err = e.ResolveString(a2ui.StringBinding("missing"), "/items/0")
	check("missing string", err, ErrNoValue)
	if err == nil || !strings.Contains(err.Error(), `"/items/0/missing"`) {
		t.Errorf("missing string: error %v does not name the path", err)
	}
	_, err = e.ResolveNumber(a2ui.NumberBinding("/name"), "")
	check("string as number", err, ErrWrongType)
	_, err = e.ResolveBoolean(a2ui.BoolBinding("/n"), "")
	check("number as boolean", err, ErrWrongType)
	_, err = e.ResolveStringList(a2ui.StringListBinding("/items"), "")
	check("objects as strings", err, ErrWrongType)
	_, err = e.ResolveStringList(a2ui.StringListBinding("/name"), "")
	check("string as list", err, ErrWrongType)
	_, err = e.ResolveValue(a2ui.ValueBinding("/missing"), "")
	check("missing value", err, ErrNoValue)
	_, err = e.ResolveBoolean(a2ui.BoolFunc(a2ui.FunctionCall{Call: "nope"}), "")
	check("unknown function", err, ErrUnknownFunction)
	_, err = e.ResolveValue(a2ui.ValueFunc(*a2ui.OpenURL(a2ui.StringLiteral("https://a2ui.org")).FunctionCall), "")
	check("action", err, ErrAction)

	deep := any("x")
	for range maxEvalDepth + 1 {
		deep = []any{deep}
	}
	_, err = e.ResolveValue(a2ui.ValueArray([]any{deep}), "")
	check("deep", err, ErrInvalidArgs)
}

func TestEvaluatorResolve(t *testing.T) {
	var e Evaluator // empty model, basic functions
	s, err := e.ResolveString(a2ui.StringFunc(a2ui.FunctionCall{
		Call: "formatNumber",
		Args: map[string]any{"value": 1234.5, "decimals": 2},
	}), "")
	if s != "1,234.50" || err != nil {
		t.Errorf("ResolveString(formatNumber) = %q, %v", s, err)
	}
	s, err = e.ResolveString(a2ui.StringFunc(a2ui.FunctionCall{Call: "required", Args: map[string]any{"value": "x"}}), "")
	if s != `{"valid":true}` || err != nil {
		t.Errorf("ResolveString(required) = %q, %v", s, err)
	}
	list, err := e.ResolveStringList(a2ui.StringListLiteral([]string{"a"}), "")
	if !reflect.DeepEqual(list, []string{"a"}) || err != nil {
		t.Errorf("ResolveStringList(literal) = %q, %v", list, err)
	}
	v, err := e.ResolveValue(a2ui.ValueArray([]any{1.0, map[string]any{"call": "not", "args": map[string]any{"value": true}}, map[string]any{"path": "/missing"}}), "")
	if want := []any{1.0, false, nil}; !reflect.DeepEqual(v, want) || err != nil {
		t.Errorf("ResolveValue(array) = %#v, %v, want %#v", v, err, want)
	}
	// A child template is an object, not a binding.
	v, err = e.ResolveValue(a2ui.ValueArray([]any{map[string]any{"path": "/x", "componentId": "row"}}), "")
	if want := []any{map[string]any{"path": "/x", "componentId": "row"}}; !reflect.DeepEqual(v, want) || err != nil {
		t.Errorf("ResolveValue(template) = %#v, %v, want %#v", v, err, want)
	}
}

func TestEvaluatorFuncs(t *testing.T) {
	var m DataModel
	m.Set("/name", "ann")
	funcs := BasicFunctions()
	funcs["upper"] = func(e *Evaluator, scope string, args map[string]any) (any, error) {
		s, ok := args["value"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: upper: value must be a string", ErrInvalidArgs)
		}
		return strings.ToUpper(s), nil
	}
	funcs["result"] = func(e *Evaluator, scope string, args map[string]any) (any, error) {
		return a2ui.ValidationResult{Valid: false, Message: "custom", Severity: a2ui.SeverityWarning}, nil
	}
	funcs["args"] = func(e *Evaluator, scope string, args map[string]any) (any, error) {
		return args, nil
	}
	funcs["nil"] = nil
	e := &Evaluator{Data: &m, Funcs: funcs}

	s, err := e.ResolveString(a2ui.FormatString(a2ui.StringLiteral("Hi ${upper(value: ${/name})}")), "")
	if s != "Hi ANN" || err != nil {
		t.Errorf("custom function in template = %q, %v", s, err)
	}
	_, err = e.ResolveString(a2ui.StringFunc(a2ui.FunctionCall{Call: "upper", Args: map[string]any{"value": 1}}), "")
	if !errors.Is(err, ErrInvalidArgs) {
		t.Errorf("custom function error = %v, want ErrInvalidArgs", err)
	}
	v, err := e.ResolveValue(a2ui.ValueFunc(a2ui.FunctionCall{Call: "result"}), "")
	if want := map[string]any{"valid": false, "message": "custom", "severity": "warning"}; !reflect.DeepEqual(v, want) || err != nil {
		t.Errorf("Go result = %#v, %v, want %#v", v, err, want)
	}
	v, err = e.ResolveValue(a2ui.ValueFunc(a2ui.FunctionCall{Call: "args", Args: map[string]any{
		"n": 3, "missing": a2ui.StringBinding("/missing"), "unset": a2ui.DynamicNumber{}, "name": a2ui.StringBinding("/name"),
	}}), "")
	if want := map[string]any{"n": 3.0, "name": "ann"}; !reflect.DeepEqual(v, want) || err != nil {
		t.Errorf("resolved args = %#v, %v, want %#v", v, err, want)
	}
	_, err = e.ResolveValue(a2ui.ValueFunc(a2ui.FunctionCall{Call: "nil"}), "")
	if !errors.Is(err, ErrUnknownFunction) {
		t.Errorf("nil function error = %v, want ErrUnknownFunction", err)
	}
}

func TestEvaluatorResolveArgs(t *testing.T) {
	e := &Evaluator{Data: evalModel(t)}
	call := *a2ui.OpenURL(a2ui.FormatString(a2ui.StringLiteral("https://example.com/${name}"))).FunctionCall
	got, err := e.ResolveArgs(call, "/items/1")
	if want := map[string]any{"url": "https://example.com/Cy"}; !reflect.DeepEqual(got, want) || err != nil {
		t.Errorf("ResolveArgs = %#v, %v, want %#v", got, err, want)
	}
	_, err = e.ResolveArgs(a2ui.FunctionCall{Call: "openUrl", Args: map[string]any{"url": a2ui.StringFunc(a2ui.FunctionCall{Call: "nope"})}}, "")
	if !errors.Is(err, ErrUnknownFunction) {
		t.Errorf("ResolveArgs error = %v, want ErrUnknownFunction", err)
	}
}

func TestEvaluatorCheck(t *testing.T) {
	e := &Evaluator{Data: evalModel(t)}
	fail := func(msg string) a2ui.ValidationResult {
		return a2ui.ValidationResult{Message: msg, Severity: a2ui.SeverityError}
	}
	fn := func(call string, args map[string]any) a2ui.DynamicValidationResult {
		return a2ui.ValidationFunc(a2ui.FunctionCall{Call: call, Args: args})
	}
	rule := func(cond a2ui.DynamicValidationResult, msg string) a2ui.CheckRule {
		return a2ui.CheckRule{Condition: cond, Message: msg}
	}
	tests := []struct {
		name    string
		checks  []a2ui.CheckRule
		want    []a2ui.ValidationResult
		wantErr error
	}{
		{"none", nil, nil, nil},
		{"pass", []a2ui.CheckRule{rule(a2ui.Required(a2ui.ValueBinding("/name")), "Name is required")}, nil, nil},
		{"fail", []a2ui.CheckRule{rule(a2ui.Email(a2ui.StringBinding("/bad")), "Enter an email address")}, []a2ui.ValidationResult{fail("Enter an email address")}, nil},
		{"default message", []a2ui.CheckRule{rule(a2ui.Email(a2ui.StringBinding("/bad")), "")}, []a2ui.ValidationResult{fail("Validation failed")}, nil},
		{"order", []a2ui.CheckRule{
			rule(a2ui.Required(a2ui.ValueBinding("/empty")), "first"),
			rule(a2ui.Required(a2ui.ValueBinding("/name")), "passes"),
			rule(fn("not", map[string]any{"value": a2ui.BoolBinding("/on")}), "third"),
		}, []a2ui.ValidationResult{fail("first"), fail("third")}, nil},
		{"boolean binding", []a2ui.CheckRule{rule(a2ui.ValidationBinding("/on"), "on"), rule(a2ui.ValidationBinding("/off"), "off")}, []a2ui.ValidationResult{fail("off")}, nil},
		{"truthy", []a2ui.CheckRule{rule(a2ui.ValidationBinding("/name"), "name"), rule(a2ui.ValidationBinding("/empty"), "empty")}, []a2ui.ValidationResult{fail("empty")}, nil},
		{"missing", []a2ui.CheckRule{rule(a2ui.ValidationBinding("/missing"), "missing")}, []a2ui.ValidationResult{fail("missing")}, nil},
		{"unset", []a2ui.CheckRule{rule(a2ui.DynamicValidationResult{}, "unset")}, []a2ui.ValidationResult{fail("unset")}, nil},
		{"error", []a2ui.CheckRule{rule(fn("nope", nil), "unknown"), rule(fn("required", map[string]any{"value": ""}), "empty")}, []a2ui.ValidationResult{fail("unknown"), fail("empty")}, ErrUnknownFunction},
	}
	for _, tt := range tests {
		got, err := e.Check(tt.checks, "")
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Check = %+v, want %+v", tt.name, got, tt.want)
		}
		if tt.wantErr == nil && err != nil || !errors.Is(err, tt.wantErr) {
			t.Errorf("%s: Check error = %v, want %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestEvaluatorCheckResult(t *testing.T) {
	var m DataModel
	m.Set("", map[string]any{
		"warn":  map[string]any{"valid": false, "message": "careful", "severity": "warning", "code": "W1"},
		"plain": map[string]any{"valid": false},
		"ok":    map[string]any{"valid": true, "message": "ignored"},
		"other": map[string]any{"x": 1},
	})
	e := &Evaluator{Data: &m}
	got, err := e.Check([]a2ui.CheckRule{
		{Condition: a2ui.ValidationBinding("/warn"), Message: "rule"},
		{Condition: a2ui.ValidationBinding("/plain"), Message: "rule"},
		{Condition: a2ui.ValidationBinding("/ok"), Message: "rule"},
		{Condition: a2ui.ValidationBinding("/other"), Message: "rule"},
	}, "")
	want := []a2ui.ValidationResult{
		{Code: "W1", Message: "careful", Severity: a2ui.SeverityWarning},
		{Message: "rule", Severity: a2ui.SeverityError},
	}
	if !reflect.DeepEqual(got, want) || err != nil {
		t.Errorf("Check = %+v, %v, want %+v", got, err, want)
	}
}
