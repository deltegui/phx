package pagination

type List[T any] struct {
	Pagination Pagination
	Items      []T
}

type Order int

const (
	OrderDescending Order = 0
	OrderAscending  Order = 1
)

type Pagination struct {
	CurrentPage     int
	ElementsPerPage int
	TotalElements   int
	Order           Order
	OrderBy         string
	Enabeld         bool
}

type Findable[ENTITY any, FILTER any] interface {
	Find(FILTER, Pagination) (List[ENTITY], error)
	FindFiltered(FILTER) (List[ENTITY], error)
	FindPaginated(Pagination) (List[ENTITY], error)
	FindFirstPage() (List[ENTITY], error)
	FindAll() ([]ENTITY, error)
	FindAllFiltered(FILTER) ([]ENTITY, error)
	FindOne(id int) (ENTITY, error)
}
