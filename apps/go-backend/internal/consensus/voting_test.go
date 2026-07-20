package consensus

import "testing"

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
			name: "a single auditor's vote is never enough on its own",
			votes: []Vote{
				{Tag: "Economic Analyst", Verdict: true},
			},
			wantResolved: false,
		},
		{
			// The defining rule of F-11: two auditors sharing the SAME tag
			// must NOT be enough, even though it's two votes. Cross-tag
			// means cross-tag.
			name: "two auditors with the same tag do not reach consensus",
			votes: []Vote{
				{Tag: "Economic Analyst", Verdict: true},
				{Tag: "Economic Analyst", Verdict: true},
			},
			wantResolved: false,
		},
		{
			name: "two auditors with different tags on the same verdict resolve it",
			votes: []Vote{
				{Tag: "Economic Analyst", Verdict: true},
				{Tag: "Geopolitical Analyst", Verdict: true},
			},
			wantVerdict:  true,
			wantResolved: true,
		},
		{
			name: "two auditors with different tags disagreeing does not resolve",
			votes: []Vote{
				{Tag: "Economic Analyst", Verdict: true},
				{Tag: "Geopolitical Analyst", Verdict: false},
			},
			wantResolved: false,
		},
		{
			// A third, dissenting auditor shouldn't block resolution once
			// two non-overlapping tags already agree — F-11 requires
			// cross-tag agreement, not unanimity.
			name: "a lone dissenter does not block an already-reached consensus",
			votes: []Vote{
				{Tag: "Economic Analyst", Verdict: true},
				{Tag: "Geopolitical Analyst", Verdict: true},
				{Tag: "Security Analyst", Verdict: false},
			},
			wantVerdict:  true,
			wantResolved: true,
		},
		{
			// The same auditor tag voting on both sides (e.g. two different
			// "Economic Analyst" auditors split on the verdict) shouldn't
			// let either side reach the two-distinct-tags bar by itself.
			name: "a split vote within a single tag resolves nothing",
			votes: []Vote{
				{Tag: "Economic Analyst", Verdict: true},
				{Tag: "Economic Analyst", Verdict: false},
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
