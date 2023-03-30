package config

import (
	"reflect"
	"testing"
)

func TestInitNavigator(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitNavigator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitNavigator() = %v, want %v", got, tt.want)
			}
		})
	}
}
