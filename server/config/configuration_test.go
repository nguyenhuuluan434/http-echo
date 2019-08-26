package config

import (
	"testing"
)

func TestLoadconfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want Configuration
	}{
		{"Test load ok", args{"./service_test.yml"}, Configuration{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Loadconfig(tt.args.path); true {
				t.Errorf("Loadconfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
