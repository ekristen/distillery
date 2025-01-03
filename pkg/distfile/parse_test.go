package distfile

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		expected    []Command
		expectError bool
	}{
		{
			name:     "Simple Distfile",
			filePath: "testdata/Distfile-simple",
			expected: []Command{
				{Action: "install", Args: []string{"ekristen/aws-nuke"}},
				{Action: "install", Args: []string{"ekristen/azure-nuke"}},
			},
		},
		{
			name:     "Nested Distfile",
			filePath: "testdata/Distfile-nested",
			expected: []Command{
				{Action: "install", Args: []string{"ekristen/aws-nuke"}},
				{Action: "install", Args: []string{"ekristen/azure-nuke"}},
				{Action: "install", Args: []string{"ekristen/cast"}},
			},
		},
		{
			name:        "Circular Include",
			filePath:    "testdata/Distfile-circular",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := Parse(tt.filePath)
			if (err != nil) != tt.expectError {
				t.Fatalf("Parse() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && !reflect.DeepEqual(commands, tt.expected) {
				t.Errorf("Parse() = %v, want %v", commands, tt.expected)
			}
		})
	}
}
