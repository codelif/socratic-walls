package main

import(
	"socraticwalls"
	"testing"
)

func Test_randomPaletteDef(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		want socraticwalls.GradientDef
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomPaletteDef()
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("randomPaletteDef() = %v, want %v", got, tt.want)
			}
		})
	}
}

