package domain

type Role struct {
	ID          string
	Name        string
	Description string
}

type Permission struct {
	ID       string
	Resource string
	Action   string
	Name     string // "<resource>:<action>"
}
