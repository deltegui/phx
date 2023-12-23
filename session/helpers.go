package session

import "github.com/deltegui/phx"

func GetUser(ctx *phx.Context) User {
	return ctx.Get(ContextKey).(User)
}

func HaveSession(ctx *phx.Context) bool {
	instance := ctx.Get(ContextKey)
	if instance == nil {
		return false
	}
	if _, ok := instance.(User); !ok {
		return false
	}
	return true
}
