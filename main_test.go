package main

import (
	"testing"
)

func TestStore(t *testing.T) {
	s := newStore()
	if s == nil {
		t.Fatal("is nil")
	}

	if err := s.fillFromFile("testdata/secrets.json"); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := s.fillFromFile("testdata/jobs.json"); err != nil {
		t.Fatalf("%+v", err)
	}

	o := s.orphans()
	if got, want := len(o.Items), 2; got != want {
		t.Fatalf("got: %d, want: %d", got, want)
	}

	for _, r := range o.Items {
		if r.Metadata.Name == "has-to-stay" {
			t.Fatalf("this resource must not be deleted")
		}
	}
}