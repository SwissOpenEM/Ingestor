package webserver

type claims struct {
	Email          string `json:"email"`
	EmailVerifierd bool   `json:"email_verified"`
}
