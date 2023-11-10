package db

import (
	"fmt"
	"log"

	"github.com/deltegui/phx/core"
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
	}
}

func NewFindableDefault[ENTITY any, FILTER any](
	db *sqlx.DB,
	buildFindSql func(filter *FILTER, params map[string]interface{}) string,
	buildOrderBySql func() string) Findable[ENTITY, FILTER] {
	return NewFindable[ENTITY, FILTER](db, buildFindSql, buildOrderBySql, 20)
}

func (repo Findable[ENTITY, FILTER]) Find(filter *FILTER, pagination *core.Pagination) (core.PaginatedList[ENTITY], error) {
	if pagination == nil {
		pagination = &core.Pagination{
			CurrentPage:     1,
			ElementsPerPage: repo.elementsPerPage,
		}
	}
	if pagination.ElementsPerPage <= 0 {
		pagination.ElementsPerPage = repo.elementsPerPage
	}
	params := map[string]interface{}{}
	sql := repo.buildFindSql(filter, params)
	count, err := repo.executeCount(sql, params)
	if err != nil {
		log.Println("Error while reading number of elements (count) for Find with pagination: ", err)
		return core.PaginatedList[ENTITY]{}, err
	}
	orderSql := repo.buildOrderBySql()
	paginationSql := repo.buildPaginationSql(*pagination, params, count)
	finalSql := fmt.Sprintf(" %s %s %s ", sql, orderSql, paginationSql)
	pagination.TotalElements = count
	log.Println(finalSql)
	log.Println(params)
	rows, err := repo.DB.NamedQuery(finalSql, params)
	if err != nil {
		log.Println("Error while executing named query (finalSql inside Find): ", err)
		return core.PaginatedList[ENTITY]{}, err
	}
	items := repo.scanRows(rows)
	return core.PaginatedList[ENTITY]{
		Items:      items,
		Pagination: *pagination,
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

func (repo Findable[ENTITY, FILTER]) buildPaginationSql(pagination core.Pagination, params map[string]interface{}, count int) string {
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
