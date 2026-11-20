package mongo

import "testing"

func TestNewClaimToken(t *testing.T) {
	first, err := newClaimToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newClaimToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("claim tokens were not unique non-empty values: %q, %q", first, second)
	}
}
