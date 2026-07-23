package consensus

import "testing"

// tagged is a small constructor so the table below reads as the scenario it
// describes rather than as struct plumbing.
func tagged(verdict bool, stake, trust float64, tags ...string) Vote {
	return Vote{Tags: tags, Verdict: verdict, Stake: stake, TrustWeight: trust}
}

func TestEvaluateCrossTagConsensus(t *testing.T) {
	tests := []struct {
		name         string
		votes        []Vote
		wantVerdict  bool
		wantResolved bool
	}{
		{
			name:         "no votes at all is not resolved",
			votes:        nil,
			wantResolved: false,
		},
		{
			name:         "a single auditor's vote is never enough on its own",
			votes:        []Vote{tagged(true, 5, 1, "Economic Analyst")},
			wantResolved: false,
		},
		{
			// The defining rule of F-11: two auditors sharing the SAME tag
			// must NOT be enough, even though it's two votes. Cross-tag means
			// cross-tag.
			name: "two auditors with the same tag do not reach consensus",
			votes: []Vote{
				tagged(true, 5, 1, "Economic Analyst"),
				tagged(true, 5, 1, "Economic Analyst"),
			},
			wantResolved: false,
		},
		{
			// Multi-tag auditors overlapping on even one tag are still not
			// independent. This is the case the old `u.tags[1]` lookup could
			// not see at all, since it only ever compared first tags — two
			// auditors who both cover "Labour Analyst" would have been read as
			// {"Economic"} vs {"Labour"} and wrongly counted as cross-tag.
			name: "multi-tag auditors overlapping on one tag is not cross-tag",
			votes: []Vote{
				tagged(true, 5, 1, "Economic Analyst", "Labour Analyst"),
				tagged(true, 5, 1, "Labour Analyst", "Housing Analyst"),
			},
			wantResolved: false,
		},
		{
			name: "two auditors with different tags resolve the claim",
			votes: []Vote{
				tagged(true, 5, 1, "Economic Analyst"),
				tagged(true, 5, 1, "Security Analyst"),
			},
			wantVerdict:  true,
			wantResolved: true,
		},
		{
			name: "multi-tag auditors with fully disjoint sets resolve",
			votes: []Vote{
				tagged(true, 5, 1, "Economic Analyst", "Labour Analyst"),
				tagged(true, 5, 1, "Climate Analyst", "Transport Analyst"),
			},
			wantVerdict:  true,
			wantResolved: true,
		},
		{
			name: "a false verdict resolves on the same terms",
			votes: []Vote{
				tagged(false, 3, 0.5, "Legal Analyst"),
				tagged(false, 3, 0.5, "Elections Analyst"),
			},
			wantVerdict:  false,
			wantResolved: true,
		},
		{
			// F-10's acceptance criterion: "outcomes are calculated using
			// auditor vote weight rather than simple majority count alone."
			// Three low-stake, zero-trust auditors are outvoted 3-2 on count
			// but outweighed on reputation.
			name: "reputation weight beats a numerical majority",
			votes: []Vote{
				tagged(true, 1, 0, "Economic Analyst"),
				tagged(true, 1, 0, "Security Analyst"),
				tagged(true, 1, 0, "Legal Analyst"),
				tagged(false, 20, 2, "Climate Analyst"),
				tagged(false, 20, 2, "Housing Analyst"),
			},
			wantVerdict:  false,
			wantResolved: true,
		},
		{
			// An untagged auditor brings no category coverage, so they can't
			// complete a cross-tag pair even though they do add weight.
			name: "an untagged auditor cannot form a cross-tag pair",
			votes: []Vote{
				tagged(true, 5, 1, "Economic Analyst"),
				tagged(true, 5, 1),
			},
			wantResolved: false,
		},
		{
			// A dead heat must not resolve — picking a side would be arbitrary.
			name: "an exact weight tie stays unresolved",
			votes: []Vote{
				tagged(true, 5, 1, "Economic Analyst"),
				tagged(true, 5, 1, "Security Analyst"),
				tagged(false, 5, 1, "Legal Analyst"),
				tagged(false, 5, 1, "Climate Analyst"),
			},
			wantResolved: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			verdict, resolved := EvaluateCrossTagConsensus(tc.votes)
			if resolved != tc.wantResolved {
				t.Fatalf("resolved = %v, want %v", resolved, tc.wantResolved)
			}
			if resolved && verdict != tc.wantVerdict {
				t.Errorf("verdict = %v, want %v", verdict, tc.wantVerdict)
			}
		})
	}
}

// Regression test for a real bug. The original implementation ranged over a
// map[bool]map[string]bool and returned the first side it found holding two
// distinct tags. When BOTH sides qualified, the winner depended on Go's
// randomised map iteration order, so identical votes could resolve `true` on
// one request and `false` on the next — a claim's verdict was effectively a
// coin flip. Repeating the same input pins that down; against the old code
// this failed within a handful of iterations.
func TestEvaluateCrossTagConsensus_IsDeterministic(t *testing.T) {
	votes := []Vote{
		tagged(true, 10, 1, "Economic Analyst"),
		tagged(true, 10, 1, "Security Analyst"),
		tagged(false, 4, 1, "Legal Analyst"),
		tagged(false, 4, 1, "Climate Analyst"),
	}

	wantVerdict, wantResolved := EvaluateCrossTagConsensus(votes)
	if !wantResolved {
		t.Fatalf("expected the heavier side to resolve")
	}
	if !wantVerdict {
		t.Fatalf("expected the heavier-weighted `true` side to win, got false")
	}

	for i := 0; i < 2000; i++ {
		verdict, resolved := EvaluateCrossTagConsensus(votes)
		if verdict != wantVerdict || resolved != wantResolved {
			t.Fatalf("iteration %d disagreed: got (%v, %v), want (%v, %v)",
				i, verdict, resolved, wantVerdict, wantResolved)
		}
	}
}

func TestDisjoint(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"fully disjoint", []string{"A"}, []string{"B"}, true},
		{"identical", []string{"A"}, []string{"A"}, false},
		{"partial overlap", []string{"A", "B"}, []string{"B", "C"}, false},
		{"disjoint multi-tag", []string{"A", "B"}, []string{"C", "D"}, true},
		{"empty left", nil, []string{"A"}, false},
		{"empty right", []string{"A"}, nil, false},
		{"both empty", nil, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := disjoint(tc.a, tc.b); got != tc.want {
				t.Errorf("disjoint(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestEffectiveWeight_NewAuditorStillCounts(t *testing.T) {
	// W_a is exactly 0 for an auditor who has never voted, so without the
	// bootstrap floor their vote would carry zero weight and a platform of
	// entirely new auditors could never resolve anything at all.
	if got := effectiveWeight(tagged(true, 5, 0)); got != 5 {
		t.Errorf("zero-trust auditor's weight = %v, want their raw stake 5", got)
	}
	if got := effectiveWeight(tagged(true, 5, 1)); got != 10 {
		t.Errorf("trust-weight-1 auditor's weight = %v, want 10", got)
	}
}
