package core

import (
	"fmt"
)

// UseCase is anything that have application domain code.
type UseCase[R any, T any] func(R) (T, error)

type Validator func(interface{}) map[string]string

type Hasher interface {
	Hash(value string) string
	Check(a, b string) bool
}

type Cypher interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

type Role int64

const (
	RoleAdmin Role = 1
	RoleUser  Role = 2
)

type UseCaseError struct {
	Code   uint16
	Reason string
}

func (caseErr UseCaseError) Error() string {
	return fmt.Sprintf("UseCaseError -> [%d] %s", caseErr.Code, caseErr.Reason)
}
