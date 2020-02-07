package jsonlogic

import (
	"bytes"
	"strings"
	"testing"
)

func TestNilRule(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	err := Apply(nil, strings.NewReader("{}"), out)
	if err == nil {
		t.Error("expected error with nil rule")
	}
}

func TestBadRule(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	err := Apply(strings.NewReader("rule"), strings.NewReader("{}"), out)
	if err == nil {
		t.Error("expected error with bad format rule")
	}
}

func TestNilData(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	err := Apply(strings.NewReader("{}"), nil, out)
	if err != nil {
		t.Error("unexpected error with nil data")
	}
}

func TestBadData(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	err := Apply(strings.NewReader("{}"), strings.NewReader("data"), out)
	if err == nil {
		t.Error("expected error with bad format data")
	}
}
