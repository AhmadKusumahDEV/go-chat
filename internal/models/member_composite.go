package models

type MemberComposite struct {
	Username string `db:"username"`
	Role     string `db:"role"`
}
