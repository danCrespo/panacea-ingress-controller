package controller

import (
	"testing"
)

func TestHelloWorld(t *testing.T) {
	t.Run("hello world test", func(t *testing.T) {
		expected := "Hello, World!"
		actual := "Hello, World!"
		if actual != expected {
			t.Errorf("expected %s, got %s", expected, actual)
		}
	})
}