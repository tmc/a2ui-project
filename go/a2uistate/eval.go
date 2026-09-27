package a2uistate

import (
	"errors"
	"fmt"
	"math"

	"github.com/a2ui-project/a2ui/go/a2ui"
)

// Evaluation failures. The errors that an [Evaluator] returns wrap one
// of these, for use with [errors.Is].
var (
	// ErrNoValue reports a dynamic value that is unset or bound to a
	// missing value, or a call to @index outside a list template.
	ErrNoValue = errors.New("a2uistate: no value")

	// ErrWrongType reports a value that is not of the type asked for,
	// such as a string where a number is needed.
	ErrWrongType = errors.New("a2uistate: wrong type")

	// ErrUnknownFunction reports a call to a function that the
	// evaluator does not have.
	ErrUnknownFunction = errors.New("a2uistate: unknown function")

	// ErrInvalidArgs reports a function call whose arguments are
	// missing, of the wrong type or malformed.
	ErrInvalidArgs = errors.New("a2uistate: invalid function arguments")

	// ErrAction reports a call, evaluated for its value, to a function
	// that performs an action and returns no value, such as openUrl.
	ErrAction = errors.New("a2uistate: action has no value")
)

// maxEvalDepth bounds the nesting of the dynamic values that an
// Evaluator resolves. It matches the web renderers.
const maxEvalDepth = 1000

// A Func implements a catalog function. It is called with the
// arguments of a call, each resolved to a JSON value: map[string]any,
// []any, string, float64, bool or nil. An argument bound to a missing
// value is absent from args. The scope is that of the call, as for
// [ResolvePath]; e evaluates the call.
//
// A Func returns a JSON value, or a value that [encoding/json] can
// marshal, such as an [a2ui.ValidationResult]. A Func that is given
// bad arguments should return an error that wraps [ErrInvalidArgs].
// A Func for an action, which has no value, should return an error
// that wraps [ErrAction]. The evaluator returns the errors of a Func
// as they are.
type Func func(e *Evaluator, scope string, args map[string]any) (any, error)

// An Evaluator resolves dynamic values against a data model, evaluating
// function calls. The zero Evaluator evaluates the basic catalog
// functions against an empty data model.
//
// An Evaluator resolves the arguments of a call before it calls the
// function, so that the logical functions do not short-circuit. It
// looks up functions by name only and ignores the catalog ID of a call.
type Evaluator struct {
	// Data is the data model. Nil is an empty data model.
	Data *DataModel

	// Funcs maps function names to their implementations. Nil means
	// the functions of [BasicFunctions]. A non-nil map replaces the
	// basic functions entirely: to add functions to them, start from
	// the map that BasicFunctions returns.
	Funcs map[string]Func
}

// ResolveValue returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating function calls. The
// dynamic values in an array are resolved too.
func (e *Evaluator) ResolveValue(d a2ui.DynamicValue, scope string) (any, error) {
	v, ok, err := e.eval(d, scope, 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, noValue(d.Binding, scope)
	}
	return v, nil
}

// ResolveString returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating function calls. A value
// that is not a string is converted as [DataModel.ResolveString] does.
func (e *Evaluator) ResolveString(d a2ui.DynamicString, scope string) (string, error) {
	v, ok, err := e.eval(d, scope, 0)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", noValue(d.Binding, scope)
	}
	return toString(v), nil
}

// ResolveNumber returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating function calls.
func (e *Evaluator) ResolveNumber(d a2ui.DynamicNumber, scope string) (float64, error) {
	v, ok, err := e.eval(d, scope, 0)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, noValue(d.Binding, scope)
	}
	n, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("%w: %s, not a number", ErrWrongType, jsonType(v))
	}
	return n, nil
}

// ResolveBoolean returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating function calls.
func (e *Evaluator) ResolveBoolean(d a2ui.DynamicBoolean, scope string) (bool, error) {
	v, ok, err := e.eval(d, scope, 0)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, noValue(d.Binding, scope)
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("%w: %s, not a boolean", ErrWrongType, jsonType(v))
	}
	return b, nil
}

// ResolveStringList returns the value of d, resolving data bindings in
// scope as [ResolvePath] does and evaluating function calls.
func (e *Evaluator) ResolveStringList(d a2ui.DynamicStringList, scope string) ([]string, error) {
	if d.Literal != nil {
		return d.Literal, nil
	}
	v, ok, err := e.eval(d, scope, 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, noValue(d.Binding, scope)
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s, not an array of strings", ErrWrongType, jsonType(v))
	}
	out := make([]string, len(list))
	for i, x := range list {
		if out[i], ok = x.(string); !ok {
			return nil, fmt.Errorf("%w: array element %d is %s, not a string", ErrWrongType, i, jsonType(x))
		}
	}
	return out, nil
}

// ResolveArgs returns the arguments of call resolved in scope as for a
// [Func], without calling the function. A renderer uses it to perform
// an action, such as the openUrl function of the basic catalog, which
// the evaluator does not perform.
func (e *Evaluator) ResolveArgs(call a2ui.FunctionCall, scope string) (map[string]any, error) {
	return e.resolveArgs(call.Args, scope, 0)
}

// Check evaluates the checks of a component in scope and returns the
// results of those that fail, in order.
//
// A condition that evaluates to a validation result object, such as
// the result of the basic catalog's required function, passes if its
// valid field is true. Any other value passes if it is truthy, as in
// JavaScript. A condition bound to a missing value fails.
//
// The message of a failed result is the message of the validation
// result, or else the message of the check, or else "Validation
// failed". Its severity is [a2ui.SeverityError] unless the validation
// result sets another.
//
// A condition that fails to evaluate fails. Check returns the results
// together with the errors of such conditions, joined by [errors.Join].
func (e *Evaluator) Check(checks []a2ui.CheckRule, scope string) ([]a2ui.ValidationResult, error) {
	var failed []a2ui.ValidationResult
	var errs []error
	for _, c := range checks {
		v, _, err := e.eval(c.Condition, scope, 0)
		if err != nil {
			errs = append(errs, err)
		}
		r := validationResult(v)
		if r.Valid {
			continue
		}
		if r.Message == "" {
			r.Message = c.Message
		}
		if r.Message == "" {
			r.Message = "Validation failed"
		}
		if r.Severity == "" {
			r.Severity = a2ui.SeverityError
		}
		failed = append(failed, r)
	}
	return failed, errors.Join(errs...)
}

// validationResult returns v as a validation result: a validation
// result object as it is and any other value by its truthiness.
func validationResult(v any) a2ui.ValidationResult {
	m, ok := v.(map[string]any)
	if !ok {
		return a2ui.ValidationResult{Valid: truthy(v)}
	}
	valid, ok := m["valid"]
	if !ok {
		return a2ui.ValidationResult{Valid: true}
	}
	r := a2ui.ValidationResult{Valid: truthy(valid)}
	if msg := m["message"]; msg != nil {
		r.Message = toString(msg)
	}
	r.Code, _ = m["code"].(string)
	if s, ok := m["severity"].(string); ok {
		r.Severity = a2ui.Severity(s)
	}
	return r
}

// truthy reports whether v is truthy in JavaScript, except that a
// validation result object is truthy if its valid field is.
func truthy(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case float64:
		return v != 0 && !math.IsNaN(v)
	case string:
		return v != ""
	case map[string]any:
		if valid, ok := v["valid"]; ok {
			return truthy(valid)
		}
	}
	return true
}

// funcs returns the functions of e.
func (e *Evaluator) funcs() map[string]Func {
	if e.Funcs == nil {
		return basicFuncs
	}
	return e.Funcs
}

// get returns the value bound to path in scope, reporting whether it
// exists.
func (e *Evaluator) get(path, scope string) (any, bool) {
	if e.Data == nil {
		return (&DataModel{}).Get(ResolvePath(path, scope))
	}
	return e.Data.Get(ResolvePath(path, scope))
}

// eval resolves the dynamic value v in scope. It reports false if v is
// unset or bound to a missing value. The value v is either a JSON
// value, in which an object with a string "path" is a data binding and
// one with a string "call" is a function call, or a Go value, such as
// an a2ui.DynamicString, which eval converts to JSON.
func (e *Evaluator) eval(v any, scope string, depth int) (any, bool, error) {
	if depth > maxEvalDepth {
		return nil, false, fmt.Errorf("%w: dynamic values nested more than %d deep", ErrInvalidArgs, maxEvalDepth)
	}
	switch v := v.(type) {
	case nil, string, float64, bool:
		return v, true, nil
	case map[string]any:
		if path, ok := v["path"].(string); ok {
			if _, ok := v["componentId"]; !ok { // not a child template
				x, ok := e.get(path, scope)
				return x, ok, nil
			}
		}
		if name, ok := v["call"].(string); ok {
			args, ok := v["args"].(map[string]any)
			if !ok && v["args"] != nil {
				return nil, false, fmt.Errorf("%w: %s: args is %s, not an object", ErrInvalidArgs, name, jsonType(v["args"]))
			}
			return e.call(name, args, scope, depth)
		}
		out := make(map[string]any, len(v))
		for k, x := range v {
			x, ok, err := e.eval(x, scope, depth+1)
			if err != nil {
				return nil, false, err
			}
			if ok {
				out[k] = x
			}
		}
		return out, true, nil
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			x, _, err := e.eval(x, scope, depth+1)
			if err != nil {
				return nil, false, err
			}
			out[i] = x
		}
		return out, true, nil
	case a2ui.DynamicString:
		switch {
		case v.Literal != nil:
			return *v.Literal, true, nil
		case v.Binding != nil:
			return e.eval(*v.Binding, scope, depth)
		case v.FunctionCall != nil:
			return e.eval(*v.FunctionCall, scope, depth)
		}
		return nil, false, nil
	case a2ui.DynamicNumber:
		switch {
		case v.Literal != nil:
			return *v.Literal, true, nil
		case v.Binding != nil:
			return e.eval(*v.Binding, scope, depth)
		case v.FunctionCall != nil:
			return e.eval(*v.FunctionCall, scope, depth)
		}
		return nil, false, nil
	case a2ui.DynamicBoolean:
		switch {
		case v.Literal != nil:
			return *v.Literal, true, nil
		case v.Binding != nil:
			return e.eval(*v.Binding, scope, depth)
		case v.FunctionCall != nil:
			return e.eval(*v.FunctionCall, scope, depth)
		}
		return nil, false, nil
	case a2ui.DynamicStringList:
		switch {
		case v.Literal != nil:
			out := make([]any, len(v.Literal))
			for i, s := range v.Literal {
				out[i] = s
			}
			return out, true, nil
		case v.Binding != nil:
			return e.eval(*v.Binding, scope, depth)
		case v.FunctionCall != nil:
			return e.eval(*v.FunctionCall, scope, depth)
		}
		return nil, false, nil
	case a2ui.DynamicValue:
		switch {
		case v.String != nil:
			return *v.String, true, nil
		case v.Number != nil:
			return *v.Number, true, nil
		case v.Bool != nil:
			return *v.Bool, true, nil
		case v.Array != nil:
			return e.eval(v.Array, scope, depth)
		case v.Binding != nil:
			return e.eval(*v.Binding, scope, depth)
		case v.FunctionCall != nil:
			return e.eval(*v.FunctionCall, scope, depth)
		}
		return nil, false, nil
	case a2ui.DynamicValidationResult:
		switch {
		case v.Binding != nil:
			return e.eval(*v.Binding, scope, depth)
		case v.FunctionCall != nil:
			return e.eval(*v.FunctionCall, scope, depth)
		}
		return nil, false, nil
	case a2ui.DataBinding:
		x, ok := e.get(v.Path, scope)
		return x, ok, nil
	case a2ui.FunctionCall:
		return e.call(v.Call, v.Args, scope, depth)
	}
	// Any other Go value, such as an int, a pointer to a dynamic value
	// or a []a2ui.DynamicBoolean, is evaluated as its JSON encoding.
	j, err := jsonValue(v)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrInvalidArgs, err)
	}
	return e.eval(j, scope, depth+1)
}

// call calls the function name with args resolved in scope.
func (e *Evaluator) call(name string, args map[string]any, scope string, depth int) (any, bool, error) {
	f, ok := e.funcs()[name]
	if !ok || f == nil {
		return nil, false, fmt.Errorf("%w %q", ErrUnknownFunction, name)
	}
	resolved, err := e.resolveArgs(args, scope, depth)
	if err != nil {
		return nil, false, err
	}
	v, err := f(e, scope, resolved)
	if err != nil {
		return nil, false, err
	}
	if v, err = jsonValue(v); err != nil {
		return nil, false, fmt.Errorf("a2uistate: result of %s: %w", name, err)
	}
	return v, true, nil
}

// resolveArgs resolves the arguments of a function call in scope,
// leaving out those bound to missing values.
func (e *Evaluator) resolveArgs(args map[string]any, scope string, depth int) (map[string]any, error) {
	out := make(map[string]any, len(args))
	for k, v := range args {
		v, ok, err := e.eval(v, scope, depth+1)
		if err != nil {
			return nil, err
		}
		if ok {
			out[k] = v
		}
	}
	return out, nil
}

// noValue returns an error wrapping ErrNoValue for a dynamic value that
// is unset or has the binding b.
func noValue(b *a2ui.DataBinding, scope string) error {
	if b != nil {
		return fmt.Errorf("%w at %q", ErrNoValue, ResolvePath(b.Path, scope))
	}
	return fmt.Errorf("%w: dynamic value is unset", ErrNoValue)
}

// jsonType returns the JSON type of v, with an article, for messages.
func jsonType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "a string"
	case float64:
		return "a number"
	case bool:
		return "a boolean"
	case []any:
		return "an array"
	case map[string]any:
		return "an object"
	}
	return fmt.Sprintf("a %T", v)
}
