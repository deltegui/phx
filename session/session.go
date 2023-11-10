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

type Id string

type User struct {
	Id   int
	Name string
	Role core.Role
}

type Entry struct {
	Id      Id
	User    User
	Timeout time.Time
}

func (entry Entry) IsValid() bool {
	return time.Now().Before(entry.Timeout)
}

type SessionStore interface {
	Save(entry Entry)
	Get(id Id) (Entry, error)
	Delete(id Id)
}

type MemoryStore struct {
	values map[Id]Entry
	mutex  sync.Mutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		values: make(map[Id]Entry),
		mutex:  sync.Mutex{},
	}
}

func (store *MemoryStore) Save(entry Entry) {
	store.mutex.Lock()
	store.values[entry.Id] = entry
	store.mutex.Unlock()
}

func (store *MemoryStore) Get(id Id) (Entry, error) {
	store.mutex.Lock()
	fmt.Println("Number of sessions: ", len(store.values))
	for key := range store.values {
		fmt.Println("Available id : ", key)
	}
	entry, ok := store.values[id]
	store.mutex.Unlock()
	if !ok {
		return Entry{}, fmt.Errorf("no session entry for id '%s'", id)
	}
	return entry, nil
}

func (store *MemoryStore) Delete(id Id) {
	store.mutex.Lock()
	delete(store.values, id)
	store.mutex.Unlock()
}

type Manager struct {
	store           SessionStore
	hasher          core.Hasher
	timeoutDuration time.Duration
}

func NewManager(store SessionStore, hasher core.Hasher, duration time.Duration) *Manager {
	return &Manager{
		store:           store,
		hasher:          hasher,
		timeoutDuration: duration,
	}
}

func NewInMemoryManager(hasher core.Hasher, duration time.Duration) *Manager {
	return NewManager(
		NewMemoryStore(),
		hasher,
		duration)
}

func (manager *Manager) Add(user User) Entry {
	id := manager.createSessionId(user)
	entry := Entry{
		Id:      id,
		User:    user,
		Timeout: time.Now().Add(manager.timeoutDuration),
	}
	manager.store.Save(entry)
	return entry
}

func (manager *Manager) createSessionId(user User) Id {
	const bits int = 32
	random, err := rand.Prime(rand.Reader, bits)
	if err != nil {
		log.Panicln("Error while creating prime number for session id: ", err)
	}
	now := time.Now().UTC().Format(time.ANSIC)
	str := fmt.Sprintf("%s-%s-%s-%d", random.String(), now, user.Name, user.Id)
	hash := manager.hasher.Hash(str)
	return Id(hash)
}

func (manager *Manager) Get(id Id) (Entry, error) {
	return manager.store.Get(id)
}

func (manager *Manager) Delete(id Id) {
	manager.store.Delete(id)
}

func (manager *Manager) GetUserIfValid(id Id) (User, error) {
	entry, err := manager.Get(id)
	if err != nil {
		return User{}, err
	}
	if entry.IsValid() {
		return entry.User, nil
	}
	manager.store.Delete(id)
	return User{}, fmt.Errorf("expired session")
}

const cookieKey string = "phx_session"

func (manager *Manager) CreateSessionCookie(w http.ResponseWriter, user User) {
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

func readSessionId(req *http.Request) (Id, *http.Cookie, error) {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return Id(""), nil, fmt.Errorf("no session cookie is present in the request")
	}
	return Id(cookie.Value), cookie, nil
}

func (manager *Manager) ReadSessionCookie(req *http.Request) (User, error) {
	sessionId, cookie, err := readSessionId(req)
	if err != nil {
		return User{}, err
	}
	if cookie.Expires.After(time.Now()) {
		return User{}, fmt.Errorf("expired sesison cookie")
	}
	user, err := manager.GetUserIfValid(sessionId)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (manager *Manager) DestroySession(w http.ResponseWriter, req *http.Request) error {
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
