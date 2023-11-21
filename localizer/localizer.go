package localizer

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/cypher"
)

type Localizer map[string]string

func (loc Localizer) Get(key string) string {
	val, ok := loc[key]
	if !ok {
		return key
	}
	return val
}

type i18n map[string]Localizer

const fallbackLanguage string = "es"

var suppoertedLangauges []string = []string{
	"es",
	"en",
}

const cookieKey string = "language"

type LocalizerStore struct {
	files     embed.FS
	sharedKey string
	errorsKey string
	cypher    core.Cypher
}

func NewLocalizerStore(files embed.FS, sharedKey, errorsKey string, cypher core.Cypher) LocalizerStore {
	return LocalizerStore{files, sharedKey, errorsKey, cypher}
}

func (ls LocalizerStore) loadFile(file string) i18n {
	raw, err := ls.files.ReadFile(file)
	if err != nil {
		log.Panicln("Error while reading file ", file, err)
	}
	var values i18n
	if err = json.Unmarshal(raw, &values); err != nil {
		log.Panicln("Error while decoding localization file ", file, err)
	}
	return values
}

func (ls LocalizerStore) GetWithoutShared(key, language string) Localizer {
	log.Println("Loading localization with key", key)
	key = fmt.Sprintf("%s.json", key)
	values := ls.loadFile(key)
	val, ok := values[language]
	if !ok {
		val, ok = values[fallbackLanguage]
		if !ok {
			log.Panicf("Failed to load fallback language ('%s') localizations for key '%s'\n", fallbackLanguage, key)
		}
	}
	return val
}

func (ls LocalizerStore) Get(key, language string) Localizer {
	loc := ls.GetWithoutShared(key, language)
	shared := ls.GetWithoutShared(ls.sharedKey, language)
	mergeLocalizers(loc, shared)
	return loc
}

func mergeLocalizers(dst, origin Localizer) {
	for key, val := range origin {
		dst[key] = val
	}
}

func (ls LocalizerStore) GetUsingRequest(key string, req *http.Request) Localizer {
	lang := ls.ReadCookie(req)
	return ls.Get(key, lang)
}

func (ls LocalizerStore) GetUsingRequestWithoutShared(key string, req *http.Request) Localizer {
	lang := ls.ReadCookie(req)
	return ls.GetWithoutShared(key, lang)
}

func (ls LocalizerStore) GetLocalizedError(err core.UseCaseError, req *http.Request) string {
	lang := ls.ReadCookie(req)
	key := strconv.Itoa(int(err.Code))
	return ls.GetWithoutShared(ls.errorsKey, lang)[key]
}

func (ls LocalizerStore) LoadIntoField(field **Localizer, key string, language string) {
	if *field == nil {
		localizer := ls.Get(key, language)
		*field = &localizer
	}
}

func (ls LocalizerStore) LoadIntoFieldUsingRequest(field **Localizer, key string, req *http.Request) {
	lang := ls.ReadCookie(req)
	ls.LoadIntoField(field, key, lang)
}

func (ls LocalizerStore) CreateCookie(w http.ResponseWriter, localization string) {
	CreateCookie(w, localization, ls.cypher)
}

func (ls LocalizerStore) ReadCookie(req *http.Request) string {
	lang, err := ReadCookie(req, ls.cypher)
	if err != nil {
		return fallbackLanguage
	}
	return lang
}

func ReadCookie(req *http.Request, cy core.Cypher) (string, error) {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return fallbackLanguage, nil
	}
	langBytes, err := cypher.DecodeCookie(cy, cookie.Value)
	if err != nil {
		return "", fmt.Errorf("cannot read language cookie: %s", err)
	}
	return string(langBytes), nil
}

func CreateCookie(w http.ResponseWriter, localization string, cy core.Cypher) error {
	lang := fallbackLanguage
	for _, supported := range suppoertedLangauges {
		if localization == supported {
			lang = localization
			break
		}
	}
	log.Printf("Creating language cookie for lang; '%s'", lang)
	encode, err := cypher.EncodeCookie(cy, lang)
	if err != nil {
		return fmt.Errorf("cannot create language cookie: %s", err)
	}
	age := 24 * time.Hour
	http.SetCookie(w, &http.Cookie{
		Name:     cookieKey,
		Value:    encode,
		Expires:  time.Now().Add(age),
		MaxAge:   int(age.Seconds()),
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
	})
	return nil
}
