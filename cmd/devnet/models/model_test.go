package models

import "testing"

func TestParameterFromArgument(t *testing.T) {
	testCases := []struct {
		argInput    string
		paramInput  string
		expectedRes string
		expectedErr error
	}{
		{"--datadir", "./dev", "--datadir=./dev", nil},
		{"--chain", "dev", "--chain=dev", nil},
		{"--dev.period", "30", "--dev.period=30", nil},
		{"", "30", "", ErrInvalidArgument},
	}

	for _, testCase := range testCases {
		got, err := ParameterFromArgument(testCase.argInput, testCase.paramInput)
		if got != testCase.expectedRes {
			t.Errorf("expected %s, got %s", testCase.expectedRes, got)
		}
		if err != testCase.expectedErr {
			t.Errorf("expected error: %s, got error: %s", testCase.expectedErr, err)
		}
	}
}
