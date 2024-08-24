package session

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/cypher"
)

type ID string

type User struct {
	ID    int64
	Name  string
	Role  core.Role
	Image string
}

type Entry struct {
	ID      ID
	User    User
	Timeout time.Time
}

func (entry Entry) IsValid() bool {
	return time.Now().Before(entry.Timeout)
}

type SessionStore interface {
	Save(entry Entry)
	Get(id ID) (Entry, error)
	Delete(id ID)
}

type MemoryStore struct {
	values map[ID]Entry
	mutex  sync.Mutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		values: make(map[ID]Entry),
		mutex:  sync.Mutex{},
	}
}

func (store *MemoryStore) Save(entry Entry) {
	store.mutex.Lock()
	store.values[entry.ID] = entry
	store.mutex.Unlock()
}

func (store *MemoryStore) Get(id ID) (Entry, error) {
	store.mutex.Lock()
	log.Println("[MemoryStore] number of sessions", len(store.values))
	for key := range store.values {
		log.Println("[MemoryStore] Available id : ", key)
	}
	entry, ok := store.values[id]
	store.mutex.Unlock()
	if !ok {
		return Entry{}, fmt.Errorf("no session entry for id '%s'", id)
	}
	return entry, nil
}

func (store *MemoryStore) Delete(id ID) {
	store.mutex.Lock()
	delete(store.values, id)
	store.mutex.Unlock()
}

type Manager struct {
	store           SessionStore
	hasher          core.Hasher
	timeoutDuration time.Duration
	cypher          core.Cypher
}

func NewManager(store SessionStore, hasher core.Hasher, duration time.Duration, cypher core.Cypher) *Manager {
	return &Manager{
		store:           store,
		hasher:          hasher,
		timeoutDuration: duration,
		cypher:          cypher,
	}
}

func NewInMemoryManager(hasher core.Hasher, duration time.Duration, cypher core.Cypher) *Manager {
	return NewManager(
		NewMemoryStore(),
		hasher,
		duration,
		cypher)
}

func (manager *Manager) Add(user User) Entry {
	id := manager.createSessionId(user)
	entry := Entry{
		ID:      id,
		User:    user,
		Timeout: time.Now().Add(manager.timeoutDuration),
	}
	manager.store.Save(entry)
	return entry
}

func (manager *Manager) createSessionId(user User) ID {
	const bits int = 32
	random, err := rand.Prime(rand.Reader, bits)
	if err != nil {
		log.Panicln("Error while creating prime number for session id: ", err)
	}
	now := time.Now().UTC().Format(time.ANSIC)
	str := fmt.Sprintf("%s-%s-%s-%d", random.String(), now, user.Name, user.ID)
	hash := manager.hasher.Hash(str)
	return ID(hash)
}

func (manager *Manager) Get(id ID) (Entry, error) {
	return manager.store.Get(id)
}

func (manager *Manager) Delete(id ID) {
	manager.store.Delete(id)
}

func (manager *Manager) GetUserIfValid(id ID) (User, error) {
	entry, err := manager.Get(id)
	if err != nil {
		return User{}, err
	}
	if entry.IsValid() {
		return entry.User, nil
	}
	manager.store.Delete(id)
	return User{}, errors.New("expired session")
}

const cookieKey string = "phx_session"

func (manager *Manager) CreateSessionCookie(w http.ResponseWriter, user User) {
	entry := manager.Add(user)
	age := core.OneDayDuration
	encoded, err := cypher.EncodeCookie(manager.cypher, string(entry.ID))
	if err != nil {
		log.Println("Cannot encrypt session cookie:", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieKey,
		Value:    encoded,
		Expires:  time.Now().Add(age),
		MaxAge:   int(age.Seconds()),
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
		HttpOnly: true,
	})
}

func readSessionId(req *http.Request, cy core.Cypher) (ID, *http.Cookie, error) {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return ID(""), nil, errors.New("no session cookie is present in the request")
	}
	id, err := cypher.DecodeCookie(cy, cookie.Value)
	if err != nil {
		return ID(""), nil, err
	}
	return ID(id), cookie, nil
}

func (manager *Manager) ReadSessionCookie(req *http.Request) (User, error) {
	sessionId, cookie, err := readSessionId(req, manager.cypher)
	if err != nil {
		return User{}, err
	}
	if cookie.Expires.After(time.Now()) {
		return User{}, errors.New("expired sesison cookie")
	}
	user, err := manager.GetUserIfValid(sessionId)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (manager *Manager) DestroySession(w http.ResponseWriter, req *http.Request) error {
	session, _, err := readSessionId(req, manager.cypher)
	if err != nil {
		return err
	}
	manager.store.Delete(session)
	http.SetCookie(w, &http.Cookie{
		Name:  cookieKey,
		Value: "",
		Path:  "/",
	})
	return nil
}
