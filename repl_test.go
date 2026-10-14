package main

import (
	"reflect"
	"testing"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input:    "Charmander Bulbasaur PIKACHU",
			expected: []string{"charmander", "bulbasaur", "pikachu"},
		},
		{
			input:    "",
			expected: []string{},
		},
		{
			input:    "   ",
			expected: []string{},
		},
		{
			input:    "Hello",
			expected: []string{"hello"},
		},
	}

	for _, c := range cases {
		actual := cleanInput(c.input)

		if len(actual) != len(c.expected) {
			t.Errorf(
				"cleanInput(%q) returned %d words; expected %d",
				c.input,
				len(actual),
				len(c.expected),
			)
			continue
		}

		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf(
				"cleanInput(%q) = %v; expected %v",
				c.input,
				actual,
				c.expected,
			)
		}
	}
}
