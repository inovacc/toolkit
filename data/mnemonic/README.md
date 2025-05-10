# Mnemonic sentence gerator [![Test](https://github.com/inovacc/mnemonic/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/inovacc/mnemonic/actions/workflows/test.yml)

## Description

This is a simple command line tool and module, to generate a mnemonic sentence. The tool is written in Go and uses the [BIP-0039](https://github.com/bitcoin/bips/blob/master/bip-0039/bip-0039-wordlists.md) wordlist.

# Go install

```bash
go install github.com/inovacc/utils/data/mnemonic@latest
```

# Import in code

```go
import "github.com/inovacc/utils/data/mnemonic"

func main() {
    // Generate a mnemonic in English with 12 words
    sentence := mnemonic.GenerateMnemonic(12, mnemonic.English)
    fmt.Println(sentence)
}
```

# Example result

```txt
[sting village youth gloom bunker omit dirt bulb update weapon sting avocado]
```

# Run mnemonic

```bash
# 1. Genera 1 mnemonic en español
mnemonic -lang=Spanish

# 2. Genera 3 mnemonics como lista de palabras
mnemonic -n 3 -f=words

# 3. Formato JSON
nemonic -f=json

# 4. Usando template
mnemonic -f=template -t="MNEMONIC: {{.Sentence}}"
```