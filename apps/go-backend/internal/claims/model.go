package claims

import "time"

type Claim struct {
	ID        string    `json:"id"`
	ArticleID string    `json:"articleId"`
	Text      string    `json:"text"`
	Tag       string    `json:"tag"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// PendingClaim is what auditors browse: a claim joined with enough article
// context to decide whether to vote on it.
type PendingClaim struct {
	ID           string `json:"id"`
	ArticleID    string `json:"articleId"`
	ArticleTitle string `json:"articleTitle"`
	Text         string `json:"text"`
	Tag          string `json:"tag"`
}
