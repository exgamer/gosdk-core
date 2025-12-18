package localizer

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"gitlab.almanit.kz/jmart/gosdk/pkg/localizer/filefinder"
	"golang.org/x/text/language"
)

var Locale *Localizer

func InitLocalizer() (*Localizer, error) {
	l := &Localizer{
		localizers: make(map[string]*i18n.Localizer),
	}

	if err := l.createBunle(); err != nil {
		return l, err
	}

	Locale = l

	return Locale, nil
}

type Localizer struct {
	bundle     *i18n.Bundle
	localizers map[string]*i18n.Localizer
}

func (l *Localizer) dynamicLocalizerSelect(lang string) (*i18n.Localizer, error) {
	var (
		localizer *i18n.Localizer
		exist     bool
	)

	localizer, exist = l.localizers[lang]

	if !exist {
		return nil, errors.New("undefined localizer by lang : " + lang)
	}

	return localizer, nil
}

func (l *Localizer) createBunle() error {
	// создаем дефолтный бандл
	l.bundle = i18n.NewBundle(language.English)
	l.bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	ff, err := filefinder.NewFileFinder()

	if err != nil {
		return err
	}

	jsons, err := ff.GetJsonFiles()

	if err != nil {
		return err
	}

	// загружаем в наш бандл файлы переводов
	for i, _ := range jsons {
		_, err = l.bundle.LoadMessageFile(jsons[i])

		if err != nil {
			return err
		}
	}

	// Создаем локализаторы
	for i, _ := range jsons {
		langCode := strings.TrimSuffix(filepath.Base(jsons[i]), filepath.Ext(jsons[i]))
		l.localizers[langCode] = i18n.NewLocalizer(l.bundle, langCode)
	}

	return nil
}
