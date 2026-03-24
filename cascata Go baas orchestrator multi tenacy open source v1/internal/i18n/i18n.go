package i18n

import (
	"encoding/json"
	"net/http"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var bundle *i18n.Bundle

// Init initializes the i18n bundle with support for PT and EN.
func Init() {
	bundle = i18n.NewBundle(language.Portuguese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	
	// Files will be loaded from the 'languages' directory
	// These will be populated in the next steps
	_ = bundle.LoadMessageFile("languages/pt-BR.json")
	_ = bundle.LoadMessageFile("languages/en-US.json")
}

// GetLocalizer returns a localizer based on the Accept-Language header.
func GetLocalizer(r *http.Request) *i18n.Localizer {
	lang := r.Header.Get("Accept-Language")
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
