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
}

func NewLocalizerStore(files embed.FS, sharedKey, errorsKey string) LocalizerStore {
	return LocalizerStore{files, sharedKey, errorsKey}
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
	lang := ReadCookie(req)
	return ls.Get(key, lang)
}

func (ls LocalizerStore) GetUsingRequestWithoutShared(key string, req *http.Request) Localizer {
	lang := ReadCookie(req)
	return ls.GetWithoutShared(key, lang)
}

func (ls LocalizerStore) GetLocalizedError(err core.UseCaseError, req *http.Request) string {
	lang := ReadCookie(req)
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
	lang := ReadCookie(req)
	ls.LoadIntoField(field, key, lang)
}

func ReadCookie(req *http.Request) string {
	cookie, err := req.Cookie(cookieKey)
	if err != nil {
		return fallbackLanguage
	}
	return cookie.Value
}

func CreateCookie(w http.ResponseWriter, localization string) {
	lang := fallbackLanguage
	for _, supported := range suppoertedLangauges {
		if localization == supported {
			lang = localization
			break
		}
	}
	log.Printf("Creating language cookie for lang; '%s'", lang)
	age := 24 * time.Hour
	http.SetCookie(w, &http.Cookie{
		Name:     cookieKey,
		Value:    lang,
		Expires:  time.Now().Add(age),
		MaxAge:   int(age.Seconds()),
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
	})
}
