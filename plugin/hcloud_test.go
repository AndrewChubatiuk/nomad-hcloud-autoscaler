package plugin

import (
	"errors"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
)

func Test_hcloudNodeIDMap(t *testing.T) {
	testCases := []struct {
		inputNode           *api.Node
		expectedOutputID    string
		expectedOutputError error
		name                string
	}{
		{
			inputNode: &api.Node{
				Attributes: map[string]string{"unique.hostname": "test-1"},
			},
			expectedOutputID:    "test-1",
			expectedOutputError: nil,
			name:                "required attribute found",
		},
		{
			inputNode: &api.Node{
				Attributes: map[string]string{},
			},
			expectedOutputID:    "",
			expectedOutputError: errors.New(`attribute "unique.hostname" not found`),
			name:                "required attribute not found",
		},
		{
			inputNode: &api.Node{
				Attributes: map[string]string{"unique.hostname": ""},
			},
			expectedOutputID:    "",
			expectedOutputError: errors.New(`attribute "unique.hostname" not found`),
			name:                "required attribute found but empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualID, actualErr := hcloudNodeIDMap(tc.inputNode)
			assert.Equal(t, tc.expectedOutputID, actualID, tc.name)
			assert.Equal(t, tc.expectedOutputError, actualErr, tc.name)
		})
	}
}

func Test_extractLabels(t *testing.T) {
	testCases := []struct {
		selector            string
		expectedOutput      map[string]string
		expectedOutputError error
		name                string
	}{
		{
			selector:            "key1=value1,key2==value2",
			expectedOutput:      nil,
			expectedOutputError: errors.New("failed to parse labels key1=value1,key2==value2"),
			name:                "extra equal sign error",
		},
		{
			selector: ",,key1=value1,,,keyN=valueN,key2=value2,",
			expectedOutput: map[string]string{
				"key1": "value1",
				"keyN": "valueN",
				"key2": "value2",
			},
			expectedOutputError: nil,
			name:                "trailing comma",
		},
		{
			selector:            "",
			expectedOutput:      map[string]string{},
			expectedOutputError: nil,
			name:                "empty selector",
		},
		{
			selector:            "asdasdasdad",
			expectedOutput:      nil,
			expectedOutputError: errors.New("failed to parse labels asdasdasdad"),
			name:                "key only error",
		},
		{
			selector:            "key1=value1,key2=value2=asdada",
			expectedOutput:      nil,
			expectedOutputError: errors.New("failed to parse labels key1=value1,key2=value2=asdada"),
			name:                "multiple equal signs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualOutput, actualErr := extractLabels(tc.selector)
			assert.Equal(t, tc.expectedOutput, actualOutput, tc.name)
			assert.Equal(t, tc.expectedOutputError, actualErr, tc.name)
		})
	}
}
