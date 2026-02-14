package kubeutils

import "testing"

func TestKubernetesUtilityFunction(t *testing.T) {
    t.Run("hello world test", func(t *testing.T) {
        if 1+1 != 2 {
            t.Errorf("Expected 1 + 1 to equal 2")
        }
    })
}