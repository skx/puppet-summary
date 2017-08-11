package main

import (
	"bytes"
	"testing"
)

func TestVersion(t *testing.T) {
	bak := out
	out = new(bytes.Buffer)
	defer func() { out = bak }()

	//
	// Expected
	//
	expected := "unreleased\n"

	s := versionCmd{}
	s.Execute(nil, nil)
	if out.(*bytes.Buffer).String() != expected {
		t.Errorf("Expected '%s' received '%s'", expected, out)
	}
}
