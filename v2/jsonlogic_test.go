package jsonlogic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Test struct {
	Rule     []byte
	Data     []byte
	Expected []byte
}

func ReadTestsFromFile() ([]Test, error) {
	var items []interface{}

	buf, err := ioutil.ReadFile("tests.json")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(buf, &items); err != nil {
		return nil, err
	}

	var tests []Test
	for _, s := range items {
		switch v := s.(type) {
		case string:
			// filter strings/comments
			continue
		case []interface{}:
			if len(v) != 3 {
				return nil, fmt.Errorf("unexpected format in tests.json, expected {rule, data, expected_result}, got len==%d", len(v))
			}
			var rule, data, expected bytes.Buffer
			if err := json.NewEncoder(&rule).Encode(v[0]); err != nil {
				return nil, fmt.Errorf("unexpected error reading rule as []byte for %v", v)
			}
			if err := json.NewEncoder(&data).Encode(v[1]); err != nil {
				return nil, fmt.Errorf("unexpected error reading data as []byte for %v", v)
			}
			if err := json.NewEncoder(&expected).Encode(v[2]); err != nil {
				return nil, fmt.Errorf("unexpected error reading expected as []byte for %v", v)
			}
			tests = append(tests, Test{rule.Bytes(), data.Bytes(), expected.Bytes()})
		default:
			return nil, fmt.Errorf("unexpected item of type %T in tests.json", v)
		}
	}

	if len(tests) < 275 {
		return nil, fmt.Errorf("unpextected length of tests: %d", len(tests))
	}

	return tests, nil
}

func TestRulesFromJsonLogic(t *testing.T) {
	tests, err := ReadTestsFromFile()
	if err != nil {
		t.Fatalf("unpexpected error reading tests from file: %v", err)
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Scenario_%d", i), func(t *testing.T) {
			var result bytes.Buffer

			err := Apply(bytes.NewBuffer(test.Rule), bytes.NewBuffer(test.Data), &result)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(result.Bytes(), test.Expected) {
				t.Errorf("expected %s, got %s", test.Expected, result.String())
			}
		})
	}
}

func BenchmarkRulesFromJsonLogic(b *testing.B) {
	tests, err := ReadTestsFromFile()
	if err != nil {
		b.Fatalf("unpexpected error reading tests from file: %v", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, test := range tests {
			var result bytes.Buffer

			err := Apply(bytes.NewBuffer(test.Rule), bytes.NewBuffer(test.Data), &result)
			if err != nil {
				b.Fatal(err)
			}

			if !reflect.DeepEqual(result.Bytes(), test.Expected) {
				b.Errorf("expected %s, got %s", test.Expected, result.String())
			}
		}
	}
}

func TestDivWithOnlyOneValue(t *testing.T) {
	rule := strings.NewReader(`{"/":[4]}`)
	data := strings.NewReader(`null`)

	var result bytes.Buffer

	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, `4`, result.String())
}

func TestSetAValue(t *testing.T) {
	rule := strings.NewReader(`{
		"map": [
			{"var": "objects"},
			{"set": [
				{"var": ""},
				"age",
				{"+": [{"var": ".age"}, 2]}
			]}
		]
	}`)

	data := strings.NewReader(`{
		"objects": [
			{"age": 100, "location": "north"},
			{"age": 500, "location": "south"}
		]
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	expected := `[
		{"age": 102, "location": "north"},
		{"age": 502, "location": "south"}
	]`

	assert.JSONEq(t, expected, result.String())
}

func TestLocalContext(t *testing.T) {
	rule := strings.NewReader(`{
		"filter": [
			{"var": "people"},
			{"==": [
				{"var": ".age"},
				{"min": {"map": [
					{"var": "people"},
					{"var": ".age"}
				]}}
			]}
		]
	}`)

	data := strings.NewReader(`{
		"people": [
			{"age":18, "name":"John"},
			{"age":20, "name":"Luke"},
			{"age":18, "name":"Mark"}
		]
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	expected := `[
		{"age": 18, "name": "John"},
		{"age": 18, "name": "Mark"}
	]`

	assert.JSONEq(t, expected, result.String())
}

func TestMapWithZeroValue(t *testing.T) {
	rule := strings.NewReader(`{
		"filter": [
			{"var": "people"},
			{"==": [
				{"var": ".age"},
				{"min": {"map": [
					{"var": "people"},
					{"var": ".age"}
				]}}
			]}
		]
	}`)

	data := strings.NewReader(`{
		"people": [
			{"age":0, "name":"John"}
		]
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	expected := `[
		{"age": 0, "name": "John"}
	]`

	assert.JSONEq(t, expected, result.String())
}

func TestListOfRanges(t *testing.T) {
	rule := strings.NewReader(`{
		"filter": [
			{"var": "people"},
			{"in": [
				{"var": ".age"},
				[
					[12, 18],
					[22, 28],
					[32, 38]
				]
			]}
		]
	}`)

	data := strings.NewReader(`{
		"people": [
			{"age":18, "name":"John"},
			{"age":20, "name":"Luke"},
			{"age":18, "name":"Mark"}
		]
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	expected := `[
		{"age": 18, "name": "John"},
		{"age": 18, "name": "Mark"}
	]`

	assert.JSONEq(t, expected, result.String())
}

func TestInSortedOperator(t *testing.T) {
	rule := strings.NewReader(`{
		"filter": [
			{"var": "people"},
			{"in_sorted": [
				{"var": ".age"},
				[
					11.00,
					[12, 14],
					[13, 18],
					2,
					"20",
					[32, 38],
					"a",
					["b", "d"]					
				]
			]}
		]
	}`)

	data := strings.NewReader(`{
		"people": [
			{"age":"18", "name":"John"},
			{"age":20, "name":"Luke"},
			{"age":18, "name":"Mark"},
			{"age":40, "name":"Donald"},
			{"age":11, "name":"Mickey"},
			{"age":"1", "name":"Minnie"},
			{"age":2, "name":"Mario"},
			{"age":"a", "name":"Mario"},
			{"age":"c", "name":"Princess"}
		]
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	expected := `[
		{"age":"18", "name": "John"},
		{"age":20, "name":"Luke"},
		{"age":18, "name": "Mark"},
		{"age":11, "name":"Mickey"},
		{"age":2, "name":"Mario"},
		{"age":"a", "name":"Mario"},
		{"age":"c", "name":"Princess"}
	]`

	assert.JSONEq(t, expected, result.String())
}

func TestSomeWithLists(t *testing.T) {
	rule := strings.NewReader(`{
		"some": [
			[511, 521, 811],
			{"in":[
				{"var":""},
				[1, 2, 3, 511]
			]}
		]
	}`)

	data := strings.NewReader(`{}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, "true", result.String())
}

func TestAllWithLists(t *testing.T) {
	rule := strings.NewReader(`{
		"all": [
			[511, 521, 811],
			{"in":[
				{"var":""},
				[511, 521, 811, 3]
			]}
		]
	}`)

	data := strings.NewReader("{}")

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, "true", result.String())
}

func TestNoneWithLists(t *testing.T) {
	rule := strings.NewReader(`{
		"none": [
			[511, 521, 811],
			{"in":[
				{"var":""},
				[1, 2]
			]}
		]
	}`)

	data := strings.NewReader("{}")

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, "true", result.String())
}

func TestInOperatorWorksWithMaps(t *testing.T) {
	rule := strings.NewReader(`{
		"some": [
			[511,521,811],
			{"in": [
				{"var": ""},
				{"map": [
					{"var": "my_list"},
					{"var": ".service_id"}
				]}
			]}
		]
	}`)

	data := strings.NewReader(`{
		"my_list": [
			{"service_id": 511},
			{"service_id": 771},
			{"service_id": 521},
			{"service_id": 181}
		]
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, "true", result.String())
}

func TestJSONLogicValidator(t *testing.T) {
	scenarios := map[string]struct {
		IsValid bool
		Rule    io.Reader
	}{
		"invalid operator": {
			IsValid: false,
			Rule:    strings.NewReader(`{"filt":[[10, 1, 100], {">=":[{"var":""},2]}]}`),
		},
		"invalid condition inside a filter": {
			IsValid: false,
			Rule:    strings.NewReader(`{"filter":[{"var":"integers"}, {"=": [{"var":""}, [10]]}]}`),
		},
		"set must be valid": {
			IsValid: true,
			Rule: strings.NewReader(`{
				"map": [
					{"var": "objects"},
					{"set": [
						{"var": ""},
						"age",
						{"+": [{"var": ".age"}, 2]}
					]}
				]
			}`),
		},
	}

	for name, scenario := range scenarios {
		t.Run(fmt.Sprintf("SCENARIO:%s", name), func(t *testing.T) {
			assert.Equal(t, scenario.IsValid, IsValid(scenario.Rule))
		})
	}
}

func TestAbsoluteValue(t *testing.T) {
	rule := strings.NewReader(`{
		"abs": { "var": "test.number" }
	}`)

	data := strings.NewReader(`{
		"test": {
			"number": -2
		}
	}`)

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, "2", result.String())
}

func TestMergeArrayOfArrays(t *testing.T) {
	rule := strings.NewReader(`{
		"merge": [
			[
				[
					"18800000",
					"18800969"
				]
			],
			[
				[
					"19840000",
					"19840969"
				]
			]
		]
	}`)
	data := strings.NewReader(`{}`)

	expectedResult := "[[\"18800000\",\"18800969\"],[\"19840000\",\"19840969\"]]"

	var result bytes.Buffer
	err := Apply(rule, data, &result)
	if err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, expectedResult, result.String())
}
