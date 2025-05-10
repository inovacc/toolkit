//go:build generate

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

type LanguageStr string

const (
	English    LanguageStr = "English"
	Spanish    LanguageStr = "Spanish"
	Korean     LanguageStr = "Korean"
	ChineseS   LanguageStr = "ChineseS"
	ChineseT   LanguageStr = "ChineseT"
	Japanese   LanguageStr = "Japanese"
	French     LanguageStr = "French"
	Czech      LanguageStr = "Czech"
	Italian    LanguageStr = "Italian"
	Portuguese LanguageStr = "Portuguese"
)

const templateFile = `package mnemonic

import "math/rand/v2"

type LanguageStr string

const (
	English    LanguageStr = "English"
	Spanish    LanguageStr = "Spanish"
	Korean     LanguageStr = "Korean"
	ChineseS   LanguageStr = "ChineseS"
	ChineseT   LanguageStr = "ChineseT"
	Japanese   LanguageStr = "Japanese"
	French     LanguageStr = "French"
	Czech      LanguageStr = "Czech"
	Italian    LanguageStr = "Italian"
	Portuguese LanguageStr = "Portuguese"
)

var wordLists = WordLists{
	words: make(map[LanguageStr][]string),
}

type WordLists struct {
	words map[LanguageStr][]string
}

func init() {
	wordLists.words[English] = []string{
		{{ .EnglishWords }},
	}
	wordLists.words[Spanish] = []string{
		{{ .SpanishWords }},
	}
	wordLists.words[Korean] = []string{
		{{ .KoreanWords }},
	}
	wordLists.words[ChineseS] = []string{
		{{ .ChineseSWords }},
	}
	wordLists.words[ChineseT] = []string{
		{{ .ChineseTWords }},
	}
	wordLists.words[Japanese] = []string{
		{{ .JapaneseWords }},
	}
	wordLists.words[French] = []string{
		{{ .FrenchWords }},
	}
	wordLists.words[Czech] = []string{
		{{ .CzechWords }},
	}
	wordLists.words[Italian] = []string{
		{{ .ItalianWords }},
	}
	wordLists.words[Portuguese] = []string{
		{{ .PortugueseWords }},
	}
}

func GetWord(lang LanguageStr, idx int) string {
	return wordLists.words[lang][idx]
}

func RandomWord(lang LanguageStr) string {
	return wordLists.words[lang][rand.IntN(len(wordLists.words[lang]))]
}

func GenerateMnemonic(size int, lang LanguageStr) []string {
	result := make([]string, size)
	for i := 0; i < size; i++ {
		result[i] = RandomWord(lang)
	}
	return result
}
`

//go:generate go run gen.go

func main() {
	urls := map[LanguageStr]string{
		English:    "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/english.txt",
		Spanish:    "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/spanish.txt",
		Japanese:   "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/japanese.txt",
		Korean:     "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/korean.txt",
		ChineseS:   "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/chinese_simplified.txt",
		ChineseT:   "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/chinese_traditional.txt",
		French:     "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/french.txt",
		Italian:    "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/italian.txt",
		Czech:      "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/czech.txt",
		Portuguese: "https://raw.githubusercontent.com/bitcoin/bips/master/bip-0039/portuguese.txt",
	}

	words := make(map[LanguageStr][]string)
	for lang, url := range urls {
		list, err := downloadFile(url)
		if err != nil {
			log.Fatalf("failed to download %s word list: %v", lang, err)
		}
		if len(list) != 2048 {
			log.Fatalf("invalid word count for %s: expected 2048, got %d", lang, len(list))
		}
		words[lang] = list
	}

	tmpl, err := template.New("wordlist").Parse(templateFile)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create("../wordlist.go")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	err = tmpl.Execute(file, map[string]string{
		"EnglishWords":    mustJoinWords(words[English]),
		"SpanishWords":    mustJoinWords(words[Spanish]),
		"KoreanWords":     mustJoinWords(words[Korean]),
		"ChineseSWords":   mustJoinWords(words[ChineseS]),
		"ChineseTWords":   mustJoinWords(words[ChineseT]),
		"JapaneseWords":   mustJoinWords(words[Japanese]),
		"FrenchWords":     mustJoinWords(words[French]),
		"CzechWords":      mustJoinWords(words[Czech]),
		"ItalianWords":    mustJoinWords(words[Italian]),
		"PortugueseWords": mustJoinWords(words[Portuguese]),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func downloadFile(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var words []string
	for scanner.Scan() {
		word := scanner.Text()
		if word != "" {
			words = append(words, fmt.Sprintf("%q", word))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return words, nil
}

func mustJoinWords(list []string) string {
	return strings.Join(list, ", ")
}
