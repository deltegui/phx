package core

import (
	"fmt"
	"time"
)

const (
	Size64 int = 64
	Size32 int = 32
	Size16 int = 16
	Size8  int = 8
)

const (
	IntBase10 int = 10
)

const OneDayDuration time.Duration = 24 * time.Hour

// UseCase is anything that have application domain code.
type UseCase[R any, T any] func(R) (T, error)

type ValidationError interface {
	Error() string
	Format(f string) string
	GetName() string
}

type Validator func(interface{}) map[string][]ValidationError

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
