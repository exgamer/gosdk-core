package localizer

import "github.com/nicksnyder/go-i18n/v2/i18n"

func (l *Localizer) GetAllMessage(key string) (map[string]string, error) {
	result := make(map[string]string)

	for lang, value := range l.localizers {
		value, err := value.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: key,
			},
		})

		if err != nil {
			continue
		}

		result[lang] = value
	}

	return result, nil
}

func (l *Localizer) GetMessageOne(key, lang string) string {
	localizer, err := l.dynamicLocalizerSelect(lang)

	if err != nil {
		return ""
	}

	value, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: key,
		},
	})

	if err != nil {
		return value
	}

	return value
}

func (l *Localizer) GetAllMessageWithTemplate(key string, template map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)

	for lang, value := range l.localizers {
		value, err := value.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: key,
			},
			TemplateData: template,
		})

		if err != nil {
			continue
		}

		result[lang] = value
	}

	return result, nil
}

func (l *Localizer) GetMessageWithTemplate(key, lang string, template map[string]interface{}) (string, error) {
	localizer, err := l.dynamicLocalizerSelect(lang)

	if err != nil {
		return "", err
	}

	value, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: key,
		},
		TemplateData: template,
	})

	if err != nil {
		return value, err
	}

	return value, nil
}
