package brotli

/* Dictionary data (words and transforms) for 1 possible context */
type encoderDictionary struct {
	words                 *dictionary
	cutoffTransformsCount uint32
	cutoffTransforms      uint64
	hashTable             []uint16
	buckets               []uint16
	dictWords             []dictWord
}

func initEncoderDictionary(dict *encoderDictionary) {
	dict.words = getDictionary()

	dict.hashTable = kStaticDictionaryHash[:]
	dict.buckets = kStaticDictionaryBuckets[:]
	dict.dictWords = kStaticDictionaryWords[:]

	dict.cutoffTransformsCount = kCutoffTransformsCount
	dict.cutoffTransforms = kCutoffTransforms
}
