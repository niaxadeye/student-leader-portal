package merch

import "strings"

var cyrTranslit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var result strings.Builder
	previousDash := false
	writeDash := func() {
		if result.Len() > 0 && !previousDash {
			result.WriteByte('-')
			previousDash = true
		}
	}
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			result.WriteRune(char)
			previousDash = false
		case cyrTranslit[char] != "":
			result.WriteString(cyrTranslit[char])
			previousDash = false
		default:
			writeDash()
		}
	}
	return strings.Trim(result.String(), "-")
}
