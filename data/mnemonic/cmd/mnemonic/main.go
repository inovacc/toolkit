package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/inovacc/toolkit/data/mnemonic"
)

var (
	count   int
	lang    string
	format  string
	tpl     string
	verbose bool
)

func init() {
	flag.IntVar(&count, "n", 1, "Number of mnemonics to generate.")
	flag.StringVar(&lang, "lang", "English", "Language for the mnemonic words.")
	flag.StringVar(&format, "f", "sentence", "Format: sentence, words, json, raw, or template.")
	flag.StringVar(&tpl, "t", "", "Go template used to format the output (used with -f=template).")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output.")
}

func main() {
	flag.Parse()

	language := mnemonic.LanguageStr(lang)
	formats := map[string]func(m *mnemonic.Mnemonic){
		"sentence": printSentence,
		"words":    printWords,
		"json":     printJSON,
		"raw":      printRaw,
		"template": printTemplate,
	}

	formatFn, ok := formats[format]
	if !ok {
		fmt.Printf("Unsupported format: %s\n", format)
		flag.PrintDefaults()
		os.Exit(1)
	}

	for i := 0; i < count; i++ {
		m, err := mnemonic.NewRandom(128, language)
		if err != nil {
			fmt.Printf("Error generating mnemonic: %v\n", err)
			os.Exit(1)
		}
		if verbose {
			fmt.Printf("Mnemonic #%d:\n", i+1)
		}
		formatFn(m)
	}
}

func printSentence(m *mnemonic.Mnemonic) {
	fmt.Println(m.Sentence())
}

func printWords(m *mnemonic.Mnemonic) {
	for _, w := range m.Words {
		fmt.Println(w)
	}
}

func printJSON(m *mnemonic.Mnemonic) {
	out := map[string]any{
		"language": m.Language,
		"words":    m.Words,
		"sentence": m.Sentence(),
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func printRaw(m *mnemonic.Mnemonic) {
	fmt.Println(strings.Join(m.Words, ","))
}

func printTemplate(m *mnemonic.Mnemonic) {
	if tpl == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Missing -t template string")
		os.Exit(1)
	}
	t := template.Must(template.New("").Parse(tpl))
	buf := &bytes.Buffer{}
	_ = t.Execute(buf, struct {
		Words    []string
		Sentence string
		Language string
	}{
		Words:    m.Words,
		Sentence: m.Sentence(),
		Language: string(m.Language),
	})
	buf.WriteByte('\n')
	fmt.Print(buf.String())
}
