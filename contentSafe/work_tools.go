package contentSafe

import (
	"github.com/go-creed/sat"
	"regexp"
	"strings"
	"unicode"
)

// ExtractChinese 提取出中文 包括繁体
func ExtractChinese(word string) string {
	var m = make([]string, 0)
	for _, runeValue := range word {
		if unicode.Is(unicode.Han, runeValue) {
			m = append(m, string(runeValue))
		}
	}
	return strings.Join(m, "")
}

// ExtractEnglish 提取出所有英文 包括大小写
func ExtractEnglish(word string) string {
	rCharacter := regexp.MustCompile("[a-zA-Z]")
	m := rCharacter.FindAllString(word, -1)
	return strings.Join(m, "")
}

// Tc2Cn 繁体转简体
func Tc2Cn(word string) string {
	return sat.DefaultDict().Read(word)
}

// PruneText 获取纯文本
func PruneText(word string) (string, string) {
	chinese := Tc2Cn(ExtractChinese(word))       // 转换成简体
	eng := strings.ToLower(ExtractEnglish(word)) // 转换成小写
	return chinese, eng
}
