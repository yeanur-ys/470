package auditors

type PendingAuditor struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	DisplayName   string   `json:"displayName"`
	CredentialURL string   `json:"credentialUrl"`
	Tags          []string `json:"tags"`
}
