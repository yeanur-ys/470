package views

// SessionView is the response shape for both login and signup — unchanged
// from the previous loginResponse, both endpoints return it.
type SessionView struct {
	Token  string `json:"token"`
	Role   string `json:"role"`
	UserID string `json:"userId"`
}
