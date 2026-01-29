package auth

import "testing"

func TestCreateSessionToken_Length(t *testing.T) {
	cases := []struct {
		inputLength int
	}{
		{8},
		{16},
		{32},
		{64},
		{128},
		{256},
	}

	for _, c := range cases {
		token, err := CreateSessionToken(c.inputLength)
		if err != nil {
			t.Fatalf("CreateSessionToken(%d) returned error: %v", c.inputLength, err)
		}

		expectedLength := 4*(c.inputLength/3) + 4 // Base64 encoding length calculation
		if len(token) != expectedLength {
			t.Errorf("CreateSessionToken(%d) = %d characters, want %d characters",
				c.inputLength, len(token), expectedLength)
		}

	}
}

func TestCreateSessionToken_Unique(t *testing.T) {
	tokens := make(map[string]bool)
	numTokens := 2500
	tokenLength := 32

	for i := 0; i < numTokens; i++ {
		token, err := CreateSessionToken(tokenLength)
		if err != nil {
			t.Fatalf("CreateSessionToken(%d) returned error: %v", tokenLength, err)
		}

		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}
