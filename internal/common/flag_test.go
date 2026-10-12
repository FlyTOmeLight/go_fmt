package common

import (
	"reflect"
	"testing"
)

func TestRepeatedStringFlags(t *testing.T) {
	var values stringSlice
	for _, value := range []string{"standard", "default;prefix(example.com)", "localmodule"} {
		if err := values.Set(value); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"standard", "default", "prefix(example.com)", "localmodule"}
	if !reflect.DeepEqual([]string(values), want) {
		t.Fatalf("got %v, want %v", values, want)
	}
}
