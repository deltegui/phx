package persistence

import "github.com/deltegui/phx/pagination"

type Findable[ENTITY any, FILTER any] interface {
	Find(FILTER, pagination.Pagination) (pagination.List[ENTITY], error)
	FindFiltered(FILTER) (pagination.List[ENTITY], error)
	FindPaginated(pagination.Pagination) (pagination.List[ENTITY], error)
	FindFirstPage() (pagination.List[ENTITY], error)
	FindAll() ([]ENTITY, error)
	FindAllFiltered(FILTER) ([]ENTITY, error)
	FindOne(id int64) (ENTITY, error)
	Exists(id int64) bool
}
