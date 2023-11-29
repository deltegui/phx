package persistence

import (
	"fmt"
	"log"
	"reflect"

	"github.com/deltegui/phx/pagination"
	"github.com/jmoiron/sqlx"
)

type SQLDao struct {
	DB *sqlx.DB
}

func NewDao(db *sqlx.DB) SQLDao {
	return SQLDao{db}
}

func (repo SQLDao) BeginOrFatal() *sqlx.Tx {
	tx, err := repo.DB.Beginx()
	if err != nil {
		log.Fatal(err)
	}
	return tx
}

type Findable[ENTITY any, FILTER any] struct {
	SQLDao
	buildFindSql    func(filter *FILTER, params map[string]interface{}) string
	buildOrderBySql func() string
	elementsPerPage int
}

func NewFindable[ENTITY any, FILTER any](
	db *sqlx.DB,
	buildFindSql func(filter *FILTER, params map[string]interface{}) string,
	buildOrderBySql func() string,
	elementsPerPage int) Findable[ENTITY, FILTER] {
	return Findable[ENTITY, FILTER]{
		SQLDao:          NewDao(db),
		buildFindSql:    buildFindSql,
		buildOrderBySql: buildOrderBySql,
		elementsPerPage: elementsPerPage,
	}
}

func NewFindableDefault[ENTITY any, FILTER any](
	db *sqlx.DB,
	buildFindSql func(filter *FILTER, params map[string]interface{}) string,
	buildOrderBySql func() string) Findable[ENTITY, FILTER] {
	return NewFindable[ENTITY, FILTER](db, buildFindSql, buildOrderBySql, 20)
}

func setFilterId[FILTER any](filter *FILTER, id int64) error {
	val := reflect.ValueOf(filter)
	val = val.Elem()
	if val.Type().Kind() != reflect.Struct {
		return fmt.Errorf("filter must be a struct")
	}
	field := val.FieldByName("Id")
	if !field.IsValid() {
		return fmt.Errorf("struct %s does not have 'Id' field", val.Type().Name())
	}
	t := field.Type()

	if t.Kind() == reflect.Ptr {
		if t.Elem().Kind() != reflect.Int {
			return fmt.Errorf("field Id must be int or *int")
		}
		if !field.CanSet() {
			return fmt.Errorf("cant set Id pointer field")
		}
		ptr := new(int64)
		*ptr = id
		field.Set(reflect.ValueOf(ptr))
		return nil
	}

	if !field.CanSet() {
		return fmt.Errorf("cant set Id int64 field")
	}
	i64 := int64(id)
	if field.OverflowInt(i64) {
		return fmt.Errorf("the type of the field Id is too small to hold the value")
	}
	field.SetInt(i64)
	return nil
}

func (repo Findable[ENTITY, FILTER]) Exists(id int64) bool {
	filter := new(FILTER)
	if err := setFilterId[FILTER](filter, id); err != nil {
		return false
	}
	list, err := repo.find(filter, &pagination.Pagination{Enabeld: false})
	if err != nil {
		return false
	}
	return len(list.Items) > 0
}

func (repo Findable[ENTITY, FILTER]) FindOne(id int64) (out ENTITY, err error) {
	filter := new(FILTER)
	if err = setFilterId[FILTER](filter, id); err != nil {
		return
	}
	list, err := repo.find(filter, &pagination.Pagination{Enabeld: false})
	if err != nil {
		return
	}
	if len(list.Items) <= 0 {
		err = fmt.Errorf("no items")
		return
	}
	return list.Items[0], nil
}

func (repo Findable[ENTITY, FILTER]) FindAllFiltered(filter FILTER) ([]ENTITY, error) {
	list, err := repo.find(&filter, &pagination.Pagination{Enabeld: false})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (repo Findable[ENTITY, FILTER]) FindAll() ([]ENTITY, error) {
	list, err := repo.find(nil, &pagination.Pagination{Enabeld: false})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (repo Findable[ENTITY, FILTER]) FindFirstPage() (pagination.List[ENTITY], error) {
	return repo.find(nil, nil)
}

func (repo Findable[ENTITY, FILTER]) FindPaginated(pag pagination.Pagination) (pagination.List[ENTITY], error) {
	return repo.find(nil, &pag)
}

func (repo Findable[ENTITY, FILTER]) FindFiltered(filter FILTER) (pagination.List[ENTITY], error) {
	return repo.find(&filter, nil)
}

func (repo Findable[ENTITY, FILTER]) Find(filter FILTER, pag pagination.Pagination) (pagination.List[ENTITY], error) {
	return repo.find(&filter, &pag)
}

func (repo Findable[ENTITY, FILTER]) find(filter *FILTER, pag *pagination.Pagination) (pagination.List[ENTITY], error) {
	if pag == nil {
		pag = &pagination.Pagination{
			CurrentPage:     1,
			ElementsPerPage: repo.elementsPerPage,
			Enabeld:         true,
		}
	}
	if pag.ElementsPerPage <= 0 {
		pag.ElementsPerPage = repo.elementsPerPage
	}
	params := map[string]interface{}{}
	sql := repo.buildFindSql(filter, params)
	count, err := repo.executeCount(sql, params)
	if err != nil {
		log.Println("Error while reading number of elements (count) for Find with pagination: ", err)
		return pagination.List[ENTITY]{}, err
	}
	orderSql := repo.buildOrderBySql()
	paginationSql := ""
	if pag.Enabeld {
		paginationSql = repo.buildPaginationSql(*pag, params, count)
	}
	finalSql := fmt.Sprintf(" %s %s %s ", sql, orderSql, paginationSql)
	pag.TotalElements = count
	log.Println(finalSql)
	log.Println(params)
	rows, err := repo.DB.NamedQuery(finalSql, params)
	if err != nil {
		log.Println("Error while executing named query (finalSql inside Find): ", err)
		return pagination.List[ENTITY]{}, err
	}
	items := repo.scanRows(rows)
	return pagination.List[ENTITY]{
		Items:      items,
		Pagination: *pag,
	}, nil
}

func (repo Findable[ENTITY, FILTER]) scanRows(rows *sqlx.Rows) []ENTITY {
	result := []ENTITY{}
	var element ENTITY
	for rows.Next() {
		err := rows.StructScan(&element)
		if err != nil {
			log.Println("Error while scanning struct (finalSql inside Find): ", err)
			return result
		}
		result = append(result, element)
	}
	log.Printf("Fetched %d elements\n", len(result))
	return result
}

func (repo Findable[ENTITY, FILTER]) buildPaginationSql(pagination pagination.Pagination, params map[string]interface{}, count int) string {
	sql := " limit :limit offset :offset "
	params["limit"] = pagination.ElementsPerPage
	params["offset"] = pagination.ElementsPerPage * (pagination.CurrentPage - 1)
	return sql
}

func (repo Findable[ENTITY, FILTER]) executeCount(sql string, params map[string]interface{}) (int, error) {
	sql = fmt.Sprintf("select count(*) from (%s) count_table", sql)
	result, err := repo.DB.NamedQuery(sql, params)
	if err != nil {
		return 0, err
	}
	var c int
	if result.Next() {
		err = result.Scan(&c)
		if err != nil {
			return 0, err
		}
	}
	return c, nil
}
