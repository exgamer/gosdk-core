package constants

const LangCodeRu string = "ru"
const LangCodeKZ string = "kk"
const LangCodeEN string = "en"

const RuLang int = 1
const KzLang int = 2
const EnLang int = 3

func GetLanguages() []int {
	return []int{
		RuLang,
		KzLang,
		EnLang,
	}
}

func GetLanguageByCode(code string) int {
	switch code {
	case LangCodeRu:
		return RuLang
	case LangCodeKZ:
		return KzLang
	case LangCodeEN:
		return EnLang
	default:
		return RuLang
	}
}

func GetLanguageCode(lang int) string {
	switch lang {
	case RuLang:
		return LangCodeRu
	case KzLang:
		return LangCodeKZ
	case EnLang:
		return LangCodeEN
	default:
		return LangCodeRu
	}
}
