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

const cookieKey string = "language"

type Store struct {
	files     embed.FS
	sharedKey string
	errorsKey string
	cypher    core.Cypher
}

func NewLocalizerStore(files embed.FS, sharedKey, errorsKey string, cypher core.Cypher) Store {
	return Store{files, sharedKey, errorsKey, cypher}
}

func (ls Store) loadFile(file string) i18n {
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

func (ls Store) GetWithoutShared(key, language string) Localizer {
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

func (ls Store) Get(key, language string) Localizer {
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

func (ls Store) GetUsingRequest(key string, req *http.Request) Localizer {
	lang := ls.ReadCookie(req)
	return ls.Get(key, lang)
}

func (ls Store) GetUsingRequestWithoutShared(key string, req *http.Request) Localizer {
	lang := ls.ReadCookie(req)
	return ls.GetWithoutShared(key, lang)
}

func (ls Store) GetLocalizedError(err core.UseCaseError, req *http.Request) string {
	lang := ls.ReadCookie(req)
	key := strconv.Itoa(int(err.Code))
	return ls.GetWithoutShared(ls.errorsKey, lang)[key]
}

func (ls Store) LoadIntoField(field **Localizer, key string, language string) {
	if *field == nil {
		localizer := ls.Get(key, language)
		*field = &localizer
	}
}

func (ls Store) LoadIntoFieldUsingRequest(field **Localizer, key string, req *http.Request) {
	lang := ls.ReadCookie(req)
	ls.LoadIntoField(field, key, lang)
}

func (ls Store) CreateCookie(w http.ResponseWriter, localization string) error {
	return CreateCookie(w, localization, ls.cypher)
}

func (ls Store) ReadCookie(req *http.Request) string {
	lang, err := ReadCookie(req, ls.cypher)
	if err != nil {
		return fallbackLanguage
	}
	return lang
}

func ReadCookie(req *http.Request, cy core.Cypher) (string, error) {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return fallbackLanguage, err
	}
	langBytes, err := cypher.DecodeCookie(cy, cookie.Value)
	if err != nil {
		return "", fmt.Errorf("cannot read language cookie: %w", err)
	}
	return langBytes, nil
}

func CreateCookie(w http.ResponseWriter, localization string, cy core.Cypher) error {
	lang := fallbackLanguage
	suppoertedLangauges := []string{
		"es",
		"en",
	}
	for _, supported := range suppoertedLangauges {
		if localization == supported {
			lang = localization
			break
		}
	}
	log.Printf("Creating language cookie for lang; '%s'", lang)
	encode, err := cypher.EncodeCookie(cy, lang)
	if err != nil {
		return fmt.Errorf("cannot create language cookie: %w", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieKey,
		Value:    encode,
		Expires:  time.Now().Add(core.OneDayDuration),
		MaxAge:   int(core.OneDayDuration.Seconds()),
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
	})
	return nil
}
