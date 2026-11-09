package labels

import (
	"reflect"
	"testing"
)

func TestNames(t *testing.T) {
	opts := Options{Enabled: true, Prefix: "trac", Fields: []string{"type", "priority", "status"}}
	got := opts.Names(map[string]string{
		"type":     "defect",
		"priority": "Critical",
		"status":   "closed",
	})
	want := []string{"trac:priority:critical", "trac:status:closed", "trac:type:defect"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names = %#v, want %#v", got, want)
	}
}

func TestSlug(t *testing.T) {
	opts := Options{Enabled: true, Prefix: "trac", Fields: []string{"component"}}
	got := opts.Names(map[string]string{"component": "Web / Frontend: Theme"})
	want := []string{"trac:component:web-frontend-theme"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names = %#v, want %#v", got, want)
	}
}

func TestDisabled(t *testing.T) {
	opts := Options{Enabled: false}
	if got := opts.Names(map[string]string{"type": "defect"}); got != nil {
		t.Fatalf("disabled Names = %#v", got)
	}
}

func TestEmptyValues(t *testing.T) {
	opts := Options{Enabled: true, Prefix: "trac", Fields: []string{"type", "priority"}}
	if got := opts.Names(map[string]string{"type": "  "}); len(got) != 0 {
		t.Fatalf("empty values = %#v", got)
	}
}
