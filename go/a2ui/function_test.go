package a2ui

import (
	"encoding/json"
	"testing"
)

func TestFunctionOptionalArgs(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"length min", Length(StringBinding("/name"), nil, new(3)), `{"call":"length","args":{"min":3,"value":{"path":"/name"}}}`},
		{"length both", Length(StringBinding("/name"), new(10), new(0)), `{"call":"length","args":{"max":10,"min":0,"value":{"path":"/name"}}}`},
		{"numeric max", Numeric(NumberBinding("/age"), new(120.0), nil), `{"call":"numeric","args":{"max":120,"value":{"path":"/age"}}}`},
		{"format number", FormatNumber(NumberLiteral(1.5), DynamicNumber{}, DynamicBoolean{}), `{"call":"formatNumber","args":{"value":1.5}}`},
		{"format number decimals", FormatNumber(NumberLiteral(1.5), NumberLiteral(2), BoolBinding("/g")), `{"call":"formatNumber","args":{"decimals":2,"grouping":{"path":"/g"},"value":1.5}}`},
		{"format currency", FormatCurrency(StringLiteral("USD"), NumberLiteral(1), DynamicNumber{}, DynamicBoolean{}), `{"call":"formatCurrency","args":{"currency":"USD","value":1}}`},
		{"pluralize", Pluralize(NumberBinding("/n"), StringLiteral("items"), DynamicString{}, DynamicString{}, StringLiteral("item"), DynamicString{}, DynamicString{}), `{"call":"pluralize","args":{"one":"item","other":"items","value":{"path":"/n"}}}`},
		{"regex", Regex(StringBinding("/zip"), `^\d{5}$`), `{"call":"regex","args":{"pattern":"^\\d{5}$","value":{"path":"/zip"}}}`},
	}
	for _, tt := range tests {
		data, err := json.Marshal(tt.v)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if string(data) != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, data, tt.want)
		}
	}
}
