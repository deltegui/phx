package session

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/deltegui/phx/core"
)

type SessionId string

type SessionUser struct {
	Id   int
	Name string
	Role core.Role
}

type SessionEntry struct {
	Id      SessionId
	User    SessionUser
	Timeout time.Time
}

func (entry SessionEntry) IsValid() bool {
	return time.Now().Before(entry.Timeout)
}

type SessionStore interface {
	Save(entry SessionEntry)
	Get(id SessionId) (SessionEntry, error)
	Delete(id SessionId)
}

type MemorySessionStore struct {
	values map[SessionId]SessionEntry
	mutex  sync.Mutex
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		values: make(map[SessionId]SessionEntry),
		mutex:  sync.Mutex{},
	}
}

func (store *MemorySessionStore) Save(entry SessionEntry) {
	store.mutex.Lock()
	store.values[entry.Id] = entry
	store.mutex.Unlock()
}

func (store *MemorySessionStore) Get(id SessionId) (SessionEntry, error) {
	store.mutex.Lock()
	fmt.Println("Number of sessions: ", len(store.values))
	for key := range store.values {
		fmt.Println("Available id : ", key)
	}
	entry, ok := store.values[id]
	store.mutex.Unlock()
	if !ok {
		return SessionEntry{}, fmt.Errorf("no session entry for id '%s'", id)
	}
	return entry, nil
}

func (store *MemorySessionStore) Delete(id SessionId) {
	store.mutex.Lock()
	delete(store.values, id)
	store.mutex.Unlock()
}

type SessionManager struct {
	store           SessionStore
	hasher          core.Hasher
	timeoutDuration time.Duration
}

func NewSessionManager(store SessionStore, hasher core.Hasher, duration time.Duration) *SessionManager {
	return &SessionManager{
		store:           store,
		hasher:          hasher,
		timeoutDuration: duration,
	}
}

func NewInMemorySessionManager(hasher core.Hasher, duration time.Duration) *SessionManager {
	return NewSessionManager(
		NewMemorySessionStore(),
		hasher,
		duration)
}

func (manager *SessionManager) Add(user SessionUser) SessionEntry {
	id := manager.createSessionId(user)
	entry := SessionEntry{
		Id:      id,
		User:    user,
		Timeout: time.Now().Add(manager.timeoutDuration),
	}
	manager.store.Save(entry)
	return entry
}

func (manager *SessionManager) createSessionId(user SessionUser) SessionId {
	const bits int = 32
	random, err := rand.Prime(rand.Reader, bits)
	if err != nil {
		log.Panicln("Error while creating prime number for session id: ", err)
	}
	now := time.Now().UTC().Format(time.ANSIC)
	str := fmt.Sprintf("%s-%s-%s-%d", random.String(), now, user.Name, user.Id)
	hash := manager.hasher.Hash(str)
	return SessionId(hash)
}

func (manager *SessionManager) Get(id SessionId) (SessionEntry, error) {
	return manager.store.Get(id)
}

func (manager *SessionManager) Delete(id SessionId) {
	manager.store.Delete(id)
}

func (manager *SessionManager) GetUserIfValid(id SessionId) (SessionUser, error) {
	entry, err := manager.Get(id)
	if err != nil {
		return SessionUser{}, err
	}
	if entry.IsValid() {
		return entry.User, nil
	}
	manager.store.Delete(id)
	return SessionUser{}, fmt.Errorf("expired session")
}

const cookieKey string = "phx_session"

func (manager *SessionManager) CreateSessionCookie(w http.ResponseWriter, user SessionUser) {
	entry := manager.Add(user)
	age := 24 * time.Hour
	http.SetCookie(w, &http.Cookie{
		Name:     cookieKey,
		Value:    string(entry.Id),
		Expires:  time.Now().Add(age),
		MaxAge:   int(age.Seconds()),
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
	})
}

func readSessionId(req *http.Request) (SessionId, *http.Cookie, error) {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return SessionId(""), nil, fmt.Errorf("no session cookie is present in the request")
	}
	return SessionId(cookie.Value), cookie, nil
}

func (manager *SessionManager) ReadSessionCookie(req *http.Request) (SessionUser, error) {
	sessionId, cookie, err := readSessionId(req)
	if err != nil {
		return SessionUser{}, err
	}
	if cookie.Expires.After(time.Now()) {
		return SessionUser{}, fmt.Errorf("expired sesison cookie")
	}
	user, err := manager.GetUserIfValid(sessionId)
	if err != nil {
		return SessionUser{}, err
	}
	return user, nil
}

func (manager *SessionManager) DestroySession(w http.ResponseWriter, req *http.Request) error {
	session, _, err := readSessionId(req)
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
