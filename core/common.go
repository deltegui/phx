package core

import "net/http"

// UseCase is anything that have application domain code.
type UseCase[R any, T any] func(R) (T, error)

type Validator func(interface{}) map[string]string

type Hasher interface {
	Hash(value string) string
	Check(a, b string) bool
}

type Cypher interface {
	Encrypt(data []byte) ([]byte, error)
	UnEncrypt(data []byte) ([]byte, error)
}

type PaginatedList[T any] struct {
	Pagination Pagination
	Items      []T
}

type PaginatedListOrder int

const (
	OrderDescending = 0
	OrderAscending  = 1
)

type Pagination struct {
	CurrentPage     int
	ElementsPerPage int
	TotalElements   int
	Order           PaginatedListOrder
	OrderBy         string
}

type Role int

const RoleAdmin Role = 1

type UseCaseError struct {
	Code   uint16
	Reason string
}

type Auth interface {
	Authorize(next http.HandlerFunc) http.HandlerFunc
	Admin(next http.HandlerFunc) http.HandlerFunc
	AuthorizeRoles(roles []Role, next http.HandlerFunc) http.HandlerFunc
}
