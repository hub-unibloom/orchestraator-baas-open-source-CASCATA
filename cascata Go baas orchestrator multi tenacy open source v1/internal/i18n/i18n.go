package i18n

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var bundle *i18n.Bundle

// Init initializes the i18n bundle with support for PT and EN.
func Init() {
	bundle = i18n.NewBundle(language.Portuguese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	
	if _, err := bundle.LoadMessageFile("languages/pt-BR.json"); err != nil {
		slog.Error("i18n: failed to load pt-BR messages", "err", err)
	}

	if _, err := bundle.LoadMessageFile("languages/en-US.json"); err != nil {
		slog.Error("i18n: failed to load en-US messages", "err", err)
	}
}

// GetLocalizer returns a localizer based on cookie or Accept-Language header.
func GetLocalizer(r *http.Request) *i18n.Localizer {
	// Priority 1: User Cookie Sovereignty
	lang := ""
	if cookie, err := r.Cookie("cascata_lang"); err == nil {
		lang = cookie.Value
	}

	// Priority 2: Browser Context fallback
	if lang == "" {
		lang = r.Header.Get("Accept-Language")
	}

	return i18n.NewLocalizer(bundle, lang)
}

// T is a helper to translate a message ID.
func T(l *i18n.Localizer, id string) string {
	msg, err := l.Localize(&i18n.LocalizeConfig{
		MessageID: id,
	})
	if err != nil {
		return id // Fallback to ID if translation fails
	}
	return msg
}
