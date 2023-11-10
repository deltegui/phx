package pgsql

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/deltegui/phx/session"
	"github.com/jmoiron/sqlx"
)

type SQLSessionStore struct {
	SQLDao
}

func NewSessionStore(db *sqlx.DB) session.SessionStore {
	return SQLSessionStore{NewDao(db)}
}

func (store SQLSessionStore) Save(entry session.SessionEntry) {
	serialized, err := json.Marshal(entry.User)
	if err != nil {
		log.Println("Error while saving session for user: ", err)
		return
	}
	insert := "insert into sessions (id, value, timeout) values ($1, $2, $3)"
	_, err = store.db.Exec(insert, entry.Id, serialized, entry.Timeout)
	if err != nil {
		fmt.Println(err)
	}
}

func (store SQLSessionStore) Get(id session.SessionId) (session.SessionEntry, error) {
	selectQuery := "select id, value, timeout from sessions where id = $1 "
	row := store.db.QueryRowx(selectQuery, id)
	dst := make(map[string]interface{})
	if err := row.MapScan(dst); err != nil {
		return session.SessionEntry{}, err
	}
	rId, ok := dst["id"]
	if !ok {
		return session.SessionEntry{}, fmt.Errorf("SQLSessionStore: Error reading id from result set in query")
	}
	sessionId := session.SessionId(rId.(string))
	rValue, ok := dst["value"]
	if !ok {
		return session.SessionEntry{}, fmt.Errorf("SQLSessionStore: Error reading id from result set in query")
	}
	var decodedUser session.SessionUser
	if err := json.Unmarshal(rValue.([]byte), &decodedUser); err != nil {
		return session.SessionEntry{}, fmt.Errorf("SQLSessionStore: Cannot decode JSON into internal.UserResopnse")
	}
	rTimeout, ok := dst["timeout"]
	if !ok {
		return session.SessionEntry{}, fmt.Errorf("SQLSessionStore: Error reading id from result set in query")
	}
	timeout, ok := rTimeout.(time.Time)
	if !ok {
		return session.SessionEntry{}, fmt.Errorf("SQLSessionStore: Cannot decode timeout")
	}
	log.Println(timeout)
	return session.SessionEntry{
		Id:      sessionId,
		User:    decodedUser,
		Timeout: timeout,
	}, nil
}

func (store SQLSessionStore) Delete(id session.SessionId) {
	delete := "delete from sessions where id = $1 "
	_, err := store.db.Exec(delete, id)
	if err != nil {
		log.Println("Error while deleting session for user: ", err)
	}
}
