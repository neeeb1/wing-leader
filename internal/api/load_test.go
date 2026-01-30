package api

import (
	"testing"
)

func TestHandleLoadLeaderboard_ValidateListLength(t *testing.T) {
	cases := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"-5", 0, true},
		{"0", 0, true},
		{"1001", 0, true},
		{"abc", 0, true},
		{"10", 10, false},
		{"500", 500, false},
		{"1000", 1000, false},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := validateListLength(c.input)

			if (err != nil) != c.wantErr {
				t.Errorf("validateListLength(%q) error = %v, wantErr %v",
					c.input, err, c.wantErr)
				return
			}

			if got != c.want {
				t.Errorf("validateListLength(%q) = %v, want %v",
					c.input, got, c.want)
			}
		})
	}
}
