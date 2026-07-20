package ranking

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestJournalistRankScore(t *testing.T) {
	tests := []struct {
		name                                                  string
		readershipVolume, verified, selfCorrected, falseClaim float64
		want                                                  float64
	}{
		{
			name: "zero readership and zero claims scores zero",
			want: 0,
		},
		{
			// R = log10(1+0) + 1*1 + 1.5*0 - 4*0 = 1
			name:     "a single verified claim adds w1",
			verified: 1,
			want:     1 * DefaultW1,
		},
		{
			// Self-correction (w2) must always outweigh plain verification
			// (w1) for the same count — this is the SRS's explicit
			// integrity incentive (F-18), not an incidental property.
			name:          "self-correction outweighs verification for an equal count",
			verified:      1,
			selfCorrected: 1,
			want:          DefaultW1 + DefaultW2,
		},
		{
			// A single false claim (w3=4) costs more than a single
			// verified claim (w1=1) earns — proven fraud must be a net
			// negative even against one offsetting success.
			name:       "a false claim costs more than one verified claim earns",
			verified:   1,
			falseClaim: 1,
			want:       DefaultW1 - DefaultW3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := JournalistRankScore(tc.readershipVolume, tc.verified, tc.selfCorrected, tc.falseClaim, DefaultW1, DefaultW2, DefaultW3)
			if !almostEqual(got, tc.want) {
				t.Errorf("JournalistRankScore() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestJournalistRankScore_LogarithmicVolumeDampener(t *testing.T) {
	// F-17: a single viral article shouldn't dominate the score linearly.
	// Going from 10 -> 10,000,000 readers (a million-fold increase) should
	// add far less than a million-fold difference in score contribution.
	low := JournalistRankScore(10, 0, 0, 0, DefaultW1, DefaultW2, DefaultW3)
	high := JournalistRankScore(10_000_000, 0, 0, 0, DefaultW1, DefaultW2, DefaultW3)

	if high-low > 10 {
		t.Errorf("expected a logarithmic (small) gap between 10 and 10,000,000 readers, got a gap of %v", high-low)
	}
	if high <= low {
		t.Errorf("expected more readers to still score at least as high, got low=%v high=%v", low, high)
	}
}

func TestCorruptionFactor(t *testing.T) {
	tests := []struct {
		name                                string
		verified, selfCorrected, falseClaim float64
		want                                float64
	}{
		{name: "no claims at all is not corrupt", want: 0},
		{name: "all false claims is fully corrupt", falseClaim: 3, want: 1},
		{name: "no false claims among others is not corrupt", verified: 2, selfCorrected: 1, want: 0},
		{name: "half false claims is half corrupt", verified: 1, falseClaim: 1, want: 0.5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CorruptionFactor(tc.verified, tc.selfCorrected, tc.falseClaim)
			if !almostEqual(got, tc.want) {
				t.Errorf("CorruptionFactor() = %v, want %v", got, tc.want)
			}
			if got < 0 || got > 1 {
				t.Errorf("CorruptionFactor() = %v, must always stay within [0,1]", got)
			}
		})
	}
}

func TestAuditorTrustWeight(t *testing.T) {
	tests := []struct {
		name                      string
		successful, failed, total float64
		want                      float64
	}{
		{name: "no votes cast at all has zero trust", want: 0},
		{
			// Wa = log10(1+5) * (1 - 0/5) = log10(6)
			name:       "all-successful votes keep full multiplier",
			successful: 5, failed: 0, total: 5,
			want: math.Log10(6),
		},
		{
			// Every vote cast against consensus should drive trust to zero
			// regardless of how many successful votes came before.
			name:       "voting against consensus every time zeroes trust out",
			successful: 5, failed: 5, total: 5,
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AuditorTrustWeight(tc.successful, tc.failed, tc.total)
			if !almostEqual(got, tc.want) {
				t.Errorf("AuditorTrustWeight() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAuditorTrustWeight_MonotonicallyPenalizesFailures(t *testing.T) {
	// FR-8/Wa: for a fixed number of successful votes, more failed votes
	// (out of a growing total) should never increase trust.
	base := AuditorTrustWeight(3, 1, 4)
	moreFailures := AuditorTrustWeight(3, 2, 5)

	if moreFailures > base {
		t.Errorf("expected trust weight to not increase as failed votes grow: base=%v moreFailures=%v", base, moreFailures)
	}
}
