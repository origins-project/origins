package gedcom

import (
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// languages are the tags for the languages 5.5.1 names in words, where 7.0 writes a
// BCP 47 tag.
var languages = map[string]model.Language{
	"afrikaans": "af", "albanian": "sq", "anglo-saxon": "ang", "catalan": "ca",
	"catalan_spn": "ca-ES", "czech": "cs", "danish": "da", "dutch": "nl", "english": "en",
	"esperanto": "eo", "estonian": "et", "faroese": "fo", "finnish": "fi", "french": "fr",
	"german": "de", "hawaiian": "haw", "hungarian": "hu", "icelandic": "is",
	"indonesian": "id", "italian": "it", "latvian": "lv", "lithuanian": "lt",
	"navaho": "nv", "norwegian": "no", "polish": "pl", "portuguese": "pt",
	"romanian": "ro", "serbo_croa": "sh", "slovak": "sk", "slovene": "sl",
	"spanish": "es", "swedish": "sv", "turkish": "tr", "wendic": "wen",
	"amharic": "am", "arabic": "ar", "armenian": "hy", "assamese": "as",
	"belorusian": "be", "bengali": "bn", "braj": "bra", "bulgarian": "bg",
	"burmese": "my", "cantonese": "yue", "church-slavic": "cu", "dogri": "doi",
	"georgian": "ka", "greek": "el", "gujarati": "gu", "hebrew": "he", "hindi": "hi",
	"japanese": "ja", "kannada": "kn", "khmer": "km", "konkani": "kok", "korean": "ko",
	"lahnda": "lah", "lao": "lo", "macedonian": "mk", "maithili": "mai",
	"malayalam": "ml", "mandrin": "cmn", "manipuri": "mni", "marathi": "mr",
	"mewari": "mtr", "nepali": "ne", "oriya": "or", "pali": "pi", "panjabi": "pa",
	"persian": "fa", "prakrit": "pra", "pusto": "ps", "rajasthani": "raj",
	"russian": "ru", "sanskrit": "sa", "serb": "sr", "tagalog": "tl", "tamil": "ta",
	"telugu": "te", "thai": "th", "tibetan": "bo", "ukrainian": "uk", "urdu": "ur",
	"vietnamese": "vi", "yiddish": "yi",
}

// language reads a LANG, a language 5.5.1 names or a 7.0 tag, marking it read. One that
// is neither is left unread, and so kept as written.
func language(node *Node) model.Language {
	word := strings.TrimSpace(node.Text())
	lang, ok := languages[strings.ToLower(word)]
	if !ok {
		var err error
		if lang, err = model.ParseLanguage(word); err != nil {
			return ""
		}
	}
	if lang != "" {
		node.Mark()
	}
	return lang
}
