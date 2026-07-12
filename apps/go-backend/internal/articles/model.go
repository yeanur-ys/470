package articles

import "time"

type Article struct {
	ID                  string    `json:"id"`
	JournalistID        string    `json:"journalistId"`
	ParentArticleID     *string   `json:"parentArticleId,omitempty"`
	Title               string    `json:"title"`
	Body                string    `json:"body"`
	Signature           string    `json:"signature"`
	ReadershipVolume    int64     `json:"readershipVolume"`
	VerifiedClaims      int       `json:"verifiedClaims"`
	SelfCorrectedClaims int       `json:"selfCorrectedClaims"`
	FalseClaims         int       `json:"falseClaims"`
	IsRetracted         bool      `json:"isRetracted"`
	CreatedAt           time.Time `json:"createdAt"`
}
