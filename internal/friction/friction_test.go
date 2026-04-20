package friction

import "testing"

func TestNewChallengeDoesNotAddWaitToImmediateChallenges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile Profile
	}{
		{name: "prompt", profile: Profile{Kind: KindPrompt}},
		{name: "passphrase", profile: Profile{Kind: KindPassphrase, Passphrase: "choose carefully"}},
		{name: "math", profile: Profile{Kind: KindMath, MathDifficulty: "medium"}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			challenge := NewChallenge(test.profile, "reason")
			if challenge.WaitSeconds != 0 {
				t.Fatalf("wait seconds = %d, want 0", challenge.WaitSeconds)
			}
		})
	}
}

func TestNewChallengePreservesWaitForWaitingChallenges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile Profile
	}{
		{name: "wait", profile: Profile{Kind: KindWait, WaitSeconds: 12}},
		{name: "combined", profile: Profile{Kind: KindCombined, Passphrase: "choose carefully", WaitSeconds: 12}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			challenge := NewChallenge(test.profile, "reason")
			if challenge.WaitSeconds != 12 {
				t.Fatalf("wait seconds = %d, want 12", challenge.WaitSeconds)
			}
		})
	}
}
