package ranking

import "math"

// Default weights: w2 > w1 rewards self-correction over baseline verification,
// w3 heavily penalizes proven false claims (SRS Section 4).
const (
	DefaultW1 = 1.0
	DefaultW2 = 1.5
	DefaultW3 = 4.0
)

// JournalistRankScore implements SRS formula (1):
// R = log10(1+V) + w1*Cvd + w2*Csc - w3*Cf
func JournalistRankScore(readershipVolume float64, verifiedClaims, selfCorrectedClaims, falseClaims float64, w1, w2, w3 float64) float64 {
	return math.Log10(1+readershipVolume) + w1*verifiedClaims + w2*selfCorrectedClaims - w3*falseClaims
}

// CorruptionFactor implements SRS formula (2), computed here server-side and
// cached; the frontend shader (node.fragment.glsl) also computes it client-side
// for smooth interpolation between updates.
func CorruptionFactor(verifiedClaims, selfCorrectedClaims, falseClaims float64) float64 {
	total := verifiedClaims + selfCorrectedClaims + falseClaims
	if total == 0 {
		return 0
	}
	return falseClaims / total
}

// AuditorTrustWeight implements SRS formula (3):
// Wa = log10(1+Vs) * (1 - Vf/Vtotal)
func AuditorTrustWeight(successfulVotes, failedVotes, totalVotes float64) float64 {
	if totalVotes == 0 {
		return 0
	}
	return math.Log10(1+successfulVotes) * (1 - failedVotes/totalVotes)
}
