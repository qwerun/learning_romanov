package utils_test

import (
	"example/utils"
	"testing"
)

func TestFoo(t *testing.T) {
	if utils.Foo() != 42 {
		defer func() {
			t.Log("DEFER")
		}()
		t.Log("expected 42")
		t.Fail()
		t.Log("Aboba")
		t.FailNow()
		t.Log("Helo")
	}
}
