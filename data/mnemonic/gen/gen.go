//go:build generate

package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
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

const (
	maxRetries     = 4                // Maximum number of retries (e.g., 3 retries means 4 attempts total)
	initialBackoff = 1 * time.Second  // Initial wait time for backoff
	maxBackoff     = 30 * time.Second // Maximum wait time for backoff
	backoffFactor  = 2                // Factor by which backoff time increases (2 for exponential)
	jitterFactor   = 0.1              // Percentage of backoff time to use as jitter (e.g., 0.1 for 10%)
)

func isRetryableError(err error, statusCode int) bool {
	if err != nil {
		// Retry on generic network errors (could be temporary)
		// Note: You might want to be more specific here, e.g., checking for net.Error and temporary/timeout
		return true
	}
	// Retry on specific HTTP status codes
	switch statusCode {
	case http.StatusTooManyRequests: // 429
		return true
	case http.StatusInternalServerError: // 500
		return true
	case http.StatusBadGateway: // 502
		return true
	case http.StatusServiceUnavailable: // 503
		return true
	case http.StatusGatewayTimeout: // 504
		return true
	default:
		return false
	}
}

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
		<-time.After(1 * time.Second)
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
	var lastErr error
	currentBackoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			jitter := time.Duration(float64(currentBackoff) * jitterFactor * (rand.Float64()*2 - 1))
			waitTime := currentBackoff + jitter
			if waitTime < 0 {
				waitTime = 0
			}

			fmt.Printf("Attempt %d for %s: Retrying in %v...\n", attempt, url, waitTime)
			time.Sleep(waitTime)

			currentBackoff *= time.Duration(backoffFactor)
			if currentBackoff > maxBackoff {
				currentBackoff = maxBackoff
			}
		}

		resp, err := http.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: http.Get failed for URL %s: %w", attempt, url, err)
			if isRetryableError(err, 0) {
				fmt.Printf("Attempt %d for %s: Encountered network error: %v. Retrying...\n", attempt, url, err)
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode == http.StatusOK {
			words := make([]string, 0)
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				word := scanner.Text()
				if word != "" {
					words = append(words, fmt.Sprintf("%q", word))
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("attempt %d: scanner error while reading from %s: %w", attempt, url, scanErr)
			}
			resp.Body.Close()
			return words, nil
		}

		statusCode := resp.StatusCode
		resp.Body.Close()

		lastErr = fmt.Errorf("attempt %d: download from %s failed with status code %d: %s", attempt, url, statusCode, resp.Status)
		if isRetryableError(nil, statusCode) {
			fmt.Printf("Attempt %d for %s: Received status %d. Retrying...\n", attempt, url, statusCode)
			continue
		}

		return nil, lastErr
	}

	return nil, fmt.Errorf("all %d retries failed for %s. Last error: %w", maxRetries, url, lastErr)
}

func mustJoinWords(list []string) string {
	return strings.Join(list, ", ")
}
