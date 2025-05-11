package brotli

import (
	"io"
	"math"
)

/* Copyright 2016 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/** Minimal value for ::BROTLI_PARAM_LGWIN parameter. */
const minWindowBits = 10

/**
 * Maximal value for ::BROTLI_PARAM_LGWIN parameter.
 *
 * @note equal to @c BROTLI_MAX_DISTANCE_BITS constant.
 */
const maxWindowBits = 24

/**
 * Maximal value for ::BROTLI_PARAM_LGWIN parameter
 * in "Large Window Brotli" (32-bit).
 */
const largeMaxWindowBits = 30

/** Minimal value for ::BROTLI_PARAM_LGBLOCK parameter. */
const minInputBlockBits = 16

/** Maximal value for ::BROTLI_PARAM_LGBLOCK parameter. */
const maxInputBlockBits = 24

/** Minimal value for ::BROTLI_PARAM_QUALITY parameter. */
const minQuality = 0

/** Maximal value for ::BROTLI_PARAM_QUALITY parameter. */
const maxQuality = 11

/** Options for ::BROTLI_PARAM_MODE parameter. */
const (
	modeGeneric = 0
	modeText    = 1
	modeFont    = 2
)

/** Default value for ::BROTLI_PARAM_QUALITY parameter. */
const defaultQuality = 11

/** Default value for ::BROTLI_PARAM_LGWIN parameter. */
const defaultWindow = 22

/** Default value for ::BROTLI_PARAM_MODE parameter. */
const defaultMode = modeGeneric

/** Operations that can be performed by streaming encoder. */
const (
	operationProcess      = 0
	operationFlush        = 1
	operationFinish       = 2
	operationEmitMetadata = 3
)

const (
	streamProcessing     = 0
	streamFlushRequested = 1
	streamFinished       = 2
	streamMetadataHead   = 3
	streamMetadataBody   = 4
)

type Writer struct {
	dst     io.Writer
	options WriterOptions
	err     error

	params           encoderParams
	hasher_          hasherHandle
	inputPos         uint64
	ringbuffer_      ringBuffer
	commands         []command
	numLiterals      uint
	lastInsertLen    uint
	lastFlushPos     uint64
	lastProcessedPos uint64
	distCache        [numDistanceShortCodes]int
	savedDistCache   [4]int
	lastBytes        uint16
	lastBytesBits    byte
	prevByte         byte
	prevByte2        byte
	storage          []byte
	smallTable       [1 << 10]int
	largeTable       []int
	largeTableSize   uint
	cmdDepths        [128]byte
	cmdBits          [128]uint16
	cmdCode          [512]byte
	cmdCodeNumbits   uint
	commandBuf       []uint32
	literalBuf       []byte
	tinyBuf          struct {
		u64 [2]uint64
		u8  [16]byte
	}
	remainingMetadataBytes uint32
	streamState            int
	isLastBlockEmitted     bool
	isInitialized          bool
}

func inputBlockSize(s *Writer) uint {
	return uint(1) << uint(s.params.lgblock)
}

func unprocessedInputSize(s *Writer) uint64 {
	return s.inputPos - s.lastProcessedPos
}

func remainingInputBlockSize(s *Writer) uint {
	var delta uint64 = unprocessedInputSize(s)
	var blockSize uint = inputBlockSize(s)
	if delta >= uint64(blockSize) {
		return 0
	}
	return blockSize - uint(delta)
}

/*
Wraps 64-bit input position to 32-bit ring-buffer position preserving

	"not-a-first-lap" feature.
*/
func wrapPosition(position uint64) uint32 {
	var result uint32 = uint32(position)
	var gb uint64 = position >> 30
	if gb > 2 {
		/* Wrap every 2GiB; The first 3GB are continuous. */
		result = result&((1<<30)-1) | (uint32((gb-1)&1)+1)<<30
	}

	return result
}

func (s *Writer) getStorage(size int) []byte {
	if len(s.storage) < size {
		s.storage = make([]byte, size)
	}

	return s.storage
}

func hashTableSize(maxTableSize uint, inputSize uint) uint {
	var htsize uint = 256
	for htsize < maxTableSize && htsize < inputSize {
		htsize <<= 1
	}

	return htsize
}

func getHashTable(s *Writer, quality int, inputSize uint, tableSize *uint) []int {
	var maxTableSize uint = maxHashTableSize(quality)
	var htsize uint = hashTableSize(maxTableSize, inputSize)
	/* Use smaller hash table when input.size() is smaller, since we
	   fill the table, incurring O(hash table size) overhead for
	   compression, and if the input is short, we won't need that
	   many hash table entries anyway. */

	var table []int
	assert(maxTableSize >= 256)
	if quality == fastOnePassCompressionQuality {
		/* Only odd shifts are supported by fast-one-pass. */
		if htsize&0xAAAAA == 0 {
			htsize <<= 1
		}
	}

	if htsize <= uint(len(s.smallTable)) {
		table = s.smallTable[:]
	} else {
		if htsize > s.largeTableSize {
			s.largeTableSize = htsize
			s.largeTable = nil
			s.largeTable = make([]int, htsize)
		}

		table = s.largeTable
	}

	*tableSize = htsize
	for i := 0; i < int(htsize); i++ {
		table[i] = 0
	}
	return table
}

func encodeWindowBits(lgwin int, largeWindow bool, lastBytes *uint16, lastBytesBits *byte) {
	if largeWindow {
		*lastBytes = uint16((lgwin&0x3F)<<8 | 0x11)
		*lastBytesBits = 14
	} else {
		if lgwin == 16 {
			*lastBytes = 0
			*lastBytesBits = 1
		} else if lgwin == 17 {
			*lastBytes = 1
			*lastBytesBits = 7
		} else if lgwin > 17 {
			*lastBytes = uint16((lgwin-17)<<1 | 0x01)
			*lastBytesBits = 4
		} else {
			*lastBytes = uint16((lgwin-8)<<4 | 0x01)
			*lastBytesBits = 7
		}
	}
}

/* Decide about the context map based on the ability of the prediction
   ability of the previous byte UTF8-prefix on the next byte. The
   prediction ability is calculated as Shannon entropy. Here we need
   Shannon entropy instead of 'BitsEntropy' since the prefix will be
   encoded with the remaining 6 bits of the following byte, and
   BitsEntropy will assume that symbol to be stored alone using Huffman
   coding. */

var kStaticContextMapContinuation = [64]uint32{
	1, 1, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}
var kStaticContextMapSimpleUTF8 = [64]uint32{
	0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

func chooseContextMap(quality int, bigramHisto []uint32, numLiteralContexts *uint, literalContextMap *[]uint32) {
	var monogramHisto = [3]uint32{0}
	var twoPrefixHisto = [6]uint32{0}
	var total uint
	var i uint
	var dummy uint
	var entropy [4]float64
	for i = 0; i < 9; i++ {
		monogramHisto[i%3] += bigramHisto[i]
		twoPrefixHisto[i%6] += bigramHisto[i]
	}

	entropy[1] = shannonEntropy(monogramHisto[:], 3, &dummy)
	entropy[2] = shannonEntropy(twoPrefixHisto[:], 3, &dummy) + shannonEntropy(twoPrefixHisto[3:], 3, &dummy)
	entropy[3] = 0
	for i = 0; i < 3; i++ {
		entropy[3] += shannonEntropy(bigramHisto[3*i:], 3, &dummy)
	}

	total = uint(monogramHisto[0] + monogramHisto[1] + monogramHisto[2])
	assert(total != 0)
	entropy[0] = 1.0 / float64(total)
	entropy[1] *= entropy[0]
	entropy[2] *= entropy[0]
	entropy[3] *= entropy[0]

	if quality < minQualityForHqContextModeling {
		/* 3 context models is a bit slower, don't use it at lower qualities. */
		entropy[3] = entropy[1] * 10
	}

	/* If expected savings by symbol are less than 0.2 bits, skip the
	   context modeling -- in exchange for faster decoding speed. */
	if entropy[1]-entropy[2] < 0.2 && entropy[1]-entropy[3] < 0.2 {
		*numLiteralContexts = 1
	} else if entropy[2]-entropy[3] < 0.02 {
		*numLiteralContexts = 2
		*literalContextMap = kStaticContextMapSimpleUTF8[:]
	} else {
		*numLiteralContexts = 3
		*literalContextMap = kStaticContextMapContinuation[:]
	}
}

/* Decide if we want to use a more complex static context map containing 13
   context values, based on the entropy reduction of histograms over the
   first 5 bits of literals. */

var kStaticContextMapComplexUTF8 = [64]uint32{
	11, 11, 12, 12, /* 0 special */
	0, 0, 0, 0, /* 4 lf */
	1, 1, 9, 9, /* 8 space */
	2, 2, 2, 2, /* !, first after space/lf and after something else. */
	1, 1, 1, 1, /* " */
	8, 3, 3, 3, /* % */
	1, 1, 1, 1, /* ({[ */
	2, 2, 2, 2, /* }]) */
	8, 4, 4, 4, /* :; */
	8, 7, 4, 4, /* . */
	8, 0, 0, 0, /* > */
	3, 3, 3, 3, /* [0..9] */
	5, 5, 10, 5, /* [A-Z] */
	5, 5, 10, 5,
	6, 6, 6, 6, /* [a-z] */
	6, 6, 6, 6,
}

func shouldUseComplexStaticContextMap(input []byte, startPos uint, length uint, mask uint, quality int, sizeHint uint, numLiteralContexts *uint, literalContextMap *[]uint32) bool {
	/* Try the more complex static context map only for long data. */
	if sizeHint < 1<<20 {
		return false
	} else {
		var endPos uint = startPos + length
		var combinedHisto = [32]uint32{0}
		var contextHisto = [13][32]uint32{[32]uint32{0}}
		var total uint32 = 0
		var entropy [3]float64
		var dummy uint
		var i uint
		var utf8Lut contextLUT = getContextLUT(contextUTF8)
		/* To make entropy calculations faster and to fit on the stack, we collect
		   histograms over the 5 most significant bits of literals. One histogram
		   without context and 13 additional histograms for each context value. */
		for ; startPos+64 <= endPos; startPos += 4096 {
			var strideEndPos uint = startPos + 64
			var prev2 byte = input[startPos&mask]
			var prev1 byte = input[(startPos+1)&mask]
			var pos uint

			/* To make the analysis of the data faster we only examine 64 byte long
			   strides at every 4kB intervals. */
			for pos = startPos + 2; pos < strideEndPos; pos++ {
				var literal byte = input[pos&mask]
				var context byte = byte(kStaticContextMapComplexUTF8[getContext(prev1, prev2, utf8Lut)])
				total++
				combinedHisto[literal>>3]++
				contextHisto[context][literal>>3]++
				prev2 = prev1
				prev1 = literal
			}
		}

		entropy[1] = shannonEntropy(combinedHisto[:], 32, &dummy)
		entropy[2] = 0
		for i = 0; i < 13; i++ {
			entropy[2] += shannonEntropy(contextHisto[i][0:], 32, &dummy)
		}

		entropy[0] = 1.0 / float64(total)
		entropy[1] *= entropy[0]
		entropy[2] *= entropy[0]

		/* The triggering heuristics below were tuned by compressing the individual
		   files of the silesia corpus. If we skip this kind of context modeling
		   for not very well compressible input (i.e. entropy using context modeling
		   is 60% of maximal entropy) or if expected savings by symbol are less
		   than 0.2 bits, then in every case when it triggers, the final compression
		   ratio is improved. Note however that this heuristics might be too strict
		   for some cases and could be tuned further. */
		if entropy[2] > 3.0 || entropy[1]-entropy[2] < 0.2 {
			return false
		} else {
			*numLiteralContexts = 13
			*literalContextMap = kStaticContextMapComplexUTF8[:]
			return true
		}
	}
}

func decideOverLiteralContextModeling(input []byte, startPos uint, length uint, mask uint, quality int, sizeHint uint, numLiteralContexts *uint, literalContextMap *[]uint32) {
	if quality < minQualityForContextModeling || length < 64 {
		return
	} else if shouldUseComplexStaticContextMap(input, startPos, length, mask, quality, sizeHint, numLiteralContexts, literalContextMap) {
	} else /* Context map was already set, nothing else to do. */
	{
		var endPos uint = startPos + length
		/* Gather bi-gram data of the UTF8 byte prefixes. To make the analysis of
		   UTF8 data faster we only examine 64 byte long strides at every 4kB
		   intervals. */

		var bigramPrefixHisto = [9]uint32{0}
		for ; startPos+64 <= endPos; startPos += 4096 {
			var lut = [4]int{0, 0, 1, 2}
			var strideEndPos uint = startPos + 64
			var prev int = lut[input[startPos&mask]>>6] * 3
			var pos uint
			for pos = startPos + 1; pos < strideEndPos; pos++ {
				var literal byte = input[pos&mask]
				bigramPrefixHisto[prev+lut[literal>>6]]++
				prev = lut[literal>>6] * 3
			}
		}

		chooseContextMap(quality, bigramPrefixHisto[0:], numLiteralContexts, literalContextMap)
	}
}

func shouldcompressEncode(data []byte, mask uint, lastFlushPos uint64, bytes uint, numLiterals uint, numCommands uint) bool {
	/* TODO: find more precise minimal block overhead. */
	if bytes <= 2 {
		return false
	}
	if numCommands < (bytes>>8)+2 {
		if float64(numLiterals) > 0.99*float64(bytes) {
			var literalHisto = [256]uint32{0}
			const kSampleRate uint32 = 13
			const kMinEntropy float64 = 7.92
			var bitCostThreshold float64 = float64(bytes) * kMinEntropy / float64(kSampleRate)
			var t uint = uint((uint32(bytes) + kSampleRate - 1) / kSampleRate)
			var pos uint32 = uint32(lastFlushPos)
			var i uint
			for i = 0; i < t; i++ {
				literalHisto[data[pos&uint32(mask)]]++
				pos += kSampleRate
			}

			if bitsEntropy(literalHisto[:], 256) > bitCostThreshold {
				return false
			}
		}
	}

	return true
}

/* Chooses the literal context mode for a metablock */
func chooseContextMode(params *encoderParams, data []byte, pos uint, mask uint, length uint) int {
	/* We only do the computation for the option of something else than
	   CONTEXT_UTF8 for the highest qualities */
	if params.quality >= minQualityForHqBlockSplitting && !isMostlyUTF8(data, pos, mask, length, kMinUTF8Ratio) {
		return contextSigned
	}

	return contextUTF8
}

func writeMetaBlockInternal(data []byte, mask uint, lastFlushPos uint64, bytes uint, isLast bool, literalContextMode int, params *encoderParams, prevByte byte, prevByte2 byte, numLiterals uint, commands []command, savedDistCache []int, distCache []int, storageIx *uint, storage []byte) {
	var wrappedLastFlushPos uint32 = wrapPosition(lastFlushPos)
	var lastBytes uint16
	var lastBytesBits byte
	var literalContextLut contextLUT = getContextLUT(literalContextMode)
	var blockParams encoderParams = *params

	if bytes == 0 {
		/* Write the ISLAST and ISEMPTY bits. */
		writeBits(2, 3, storageIx, storage)

		*storageIx = (*storageIx + 7) &^ 7
		return
	}

	if !shouldcompressEncode(data, mask, lastFlushPos, bytes, numLiterals, uint(len(commands))) {
		/* Restore the distance cache, as its last update by
		   CreateBackwardReferences is now unused. */
		copy(distCache, savedDistCache[:4])

		storeUncompressedMetaBlock(isLast, data, uint(wrappedLastFlushPos), mask, bytes, storageIx, storage)
		return
	}

	assert(*storageIx <= 14)
	lastBytes = uint16(storage[1])<<8 | uint16(storage[0])
	lastBytesBits = byte(*storageIx)
	if params.quality <= maxQualityForStaticEntropyCodes {
		storeMetaBlockFast(data, uint(wrappedLastFlushPos), bytes, mask, isLast, params, commands, storageIx, storage)
	} else if params.quality < minQualityForBlockSplit {
		storeMetaBlockTrivial(data, uint(wrappedLastFlushPos), bytes, mask, isLast, params, commands, storageIx, storage)
	} else {
		mb := getMetaBlockSplit()
		if params.quality < minQualityForHqBlockSplitting {
			var numLiteralContexts uint = 1
			var literalContextMap []uint32 = nil
			if !params.disableLiteralContextModeling {
				decideOverLiteralContextModeling(data, uint(wrappedLastFlushPos), bytes, mask, params.quality, params.sizeHint, &numLiteralContexts, &literalContextMap)
			}

			buildMetaBlockGreedy(data, uint(wrappedLastFlushPos), mask, prevByte, prevByte2, literalContextLut, numLiteralContexts, literalContextMap, commands, mb)
		} else {
			buildMetaBlock(data, uint(wrappedLastFlushPos), mask, &blockParams, prevByte, prevByte2, commands, literalContextMode, mb)
		}

		if params.quality >= minQualityForOptimizeHistograms {
			/* The number of distance symbols effectively used for distance
			   histograms. It might be less than distance alphabet size
			   for "Large Window Brotli" (32-bit). */
			var numEffectiveDistCodes uint32 = blockParams.dist.alphabetSize
			if numEffectiveDistCodes > numHistogramDistanceSymbols {
				numEffectiveDistCodes = numHistogramDistanceSymbols
			}

			optimizeHistograms(numEffectiveDistCodes, mb)
		}

		storeMetaBlock(data, uint(wrappedLastFlushPos), bytes, mask, prevByte, prevByte2, isLast, &blockParams, literalContextMode, commands, mb, storageIx, storage)
		freeMetaBlockSplit(mb)
	}

	if bytes+4 < *storageIx>>3 {
		/* Restore the distance cache and last byte. */
		copy(distCache, savedDistCache[:4])

		storage[0] = byte(lastBytes)
		storage[1] = byte(lastBytes >> 8)
		*storageIx = uint(lastBytesBits)
		storeUncompressedMetaBlock(isLast, data, uint(wrappedLastFlushPos), mask, bytes, storageIx, storage)
	}
}

func chooseDistanceParams(params *encoderParams) {
	var distancePostfixBits uint32 = 0
	var num_direct_distance_codes uint32 = 0

	if params.quality >= minQualityForNonzeroDistanceParams {
		var ndirectMsb uint32
		if params.mode == modeFont {
			distancePostfixBits = 1
			num_direct_distance_codes = 12
		} else {
			distancePostfixBits = params.dist.distancePostfixBits
			num_direct_distance_codes = params.dist.numDirectDistanceCodes
		}

		ndirectMsb = (num_direct_distance_codes >> distancePostfixBits) & 0x0F
		if distancePostfixBits > maxNpostfix || num_direct_distance_codes > maxNdirect || ndirectMsb<<distancePostfixBits != num_direct_distance_codes {
			distancePostfixBits = 0
			num_direct_distance_codes = 0
		}
	}

	initDistanceParams(params, distancePostfixBits, num_direct_distance_codes)
}

func ensureInitialized(s *Writer) bool {
	if s.isInitialized {
		return true
	}

	s.lastBytesBits = 0
	s.lastBytes = 0
	s.remainingMetadataBytes = math.MaxUint32

	sanitizeParams(&s.params)
	s.params.lgblock = computeLgBlock(&s.params)
	chooseDistanceParams(&s.params)

	ringBufferSetup(&s.params, &s.ringbuffer_)

	/* Initialize last byte with stream header. */
	{
		var lgwin int = int(s.params.lgwin)
		if s.params.quality == fastOnePassCompressionQuality || s.params.quality == fastTwoPassCompressionQuality {
			lgwin = brotliMaxInt(lgwin, 18)
		}

		encodeWindowBits(lgwin, s.params.largeWindow, &s.lastBytes, &s.lastBytesBits)
	}

	if s.params.quality == fastOnePassCompressionQuality {
		s.cmdDepths = [128]byte{
			0, 4, 4, 5, 6, 6, 7, 7, 7, 7, 7, 8, 8, 8, 8, 8,
			0, 0, 0, 4, 4, 4, 4, 4, 5, 5, 6, 6, 6, 6, 7, 7,
			7, 7, 10, 10, 10, 10, 10, 10, 0, 4, 4, 5, 5, 5, 6, 6,
			7, 8, 8, 9, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
			5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 5, 5, 4, 4, 4, 4,
			4, 4, 4, 5, 5, 5, 5, 5, 5, 6, 6, 7, 7, 7, 8, 10,
			12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
		}
		s.cmdBits = [128]uint16{
			0, 0, 8, 9, 3, 35, 7, 71,
			39, 103, 23, 47, 175, 111, 239, 31,
			0, 0, 0, 4, 12, 2, 10, 6,
			13, 29, 11, 43, 27, 59, 87, 55,
			15, 79, 319, 831, 191, 703, 447, 959,
			0, 14, 1, 25, 5, 21, 19, 51,
			119, 159, 95, 223, 479, 991, 63, 575,
			127, 639, 383, 895, 255, 767, 511, 1023,
			14, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			27, 59, 7, 39, 23, 55, 30, 1, 17, 9, 25, 5, 0, 8, 4, 12,
			2, 10, 6, 21, 13, 29, 3, 19, 11, 15, 47, 31, 95, 63, 127, 255,
			767, 2815, 1791, 3839, 511, 2559, 1535, 3583, 1023, 3071, 2047, 4095,
		}
		s.cmdCode = [512]byte{
			0xff, 0x77, 0xd5, 0xbf, 0xe7, 0xde, 0xea, 0x9e, 0x51, 0x5d, 0xde, 0xc6,
			0x70, 0x57, 0xbc, 0x58, 0x58, 0x58, 0xd8, 0xd8, 0x58, 0xd5, 0xcb, 0x8c,
			0xea, 0xe0, 0xc3, 0x87, 0x1f, 0x83, 0xc1, 0x60, 0x1c, 0x67, 0xb2, 0xaa,
			0x06, 0x83, 0xc1, 0x60, 0x30, 0x18, 0xcc, 0xa1, 0xce, 0x88, 0x54, 0x94,
			0x46, 0xe1, 0xb0, 0xd0, 0x4e, 0xb2, 0xf7, 0x04, 0x00,
		}
		s.cmdCodeNumbits = 448
	}

	s.isInitialized = true
	return true
}

func encoderInitParams(params *encoderParams) {
	params.mode = defaultMode
	params.largeWindow = false
	params.quality = defaultQuality
	params.lgwin = defaultWindow
	params.lgblock = 0
	params.sizeHint = 0
	params.disableLiteralContextModeling = false
	initEncoderDictionary(&params.dictionary)
	params.dist.distancePostfixBits = 0
	params.dist.numDirectDistanceCodes = 0
	params.dist.alphabetSize = uint32(distanceAlphabetSize(0, 0, maxDistanceBits))
	params.dist.maxDistance = maxDistanceConst
}

func encoderInitState(s *Writer) {
	encoderInitParams(&s.params)
	s.inputPos = 0
	s.commands = s.commands[:0]
	s.numLiterals = 0
	s.lastInsertLen = 0
	s.lastFlushPos = 0
	s.lastProcessedPos = 0
	s.prevByte = 0
	s.prevByte2 = 0
	if s.hasher_ != nil {
		s.hasher_.Common().is_prepared_ = false
	}
	s.cmdCodeNumbits = 0
	s.streamState = streamProcessing
	s.isLastBlockEmitted = false
	s.isInitialized = false

	ringBufferInit(&s.ringbuffer_)

	/* Initialize distance cache. */
	s.distCache[0] = 4

	s.distCache[1] = 11
	s.distCache[2] = 15
	s.distCache[3] = 16

	/* Save the state of the distance cache in case we need to restore it for
	   emitting an uncompressed block. */
	copy(s.savedDistCache[:], s.distCache[:])
}

/*
Copies the given input data to the internal ring buffer of the compressor.
No processing of the data occurs at this time and this function can be
called multiple times before calling WriteBrotliData() to process the
accumulated input. At most input_block_size() bytes of input data can be
copied to the ring buffer, otherwise the next WriteBrotliData() will fail.
*/
func copyInputToRingBuffer(s *Writer, inputSize uint, inputBuffer []byte) {
	var ringbuffer_ *ringBuffer = &s.ringbuffer_
	ringBufferWrite(inputBuffer, inputSize, ringbuffer_)
	s.inputPos += uint64(inputSize)

	/* TL;DR: If needed, initialize 7 more bytes in the ring buffer to make the
	   hashing not depend on uninitialized data. This makes compression
	   deterministic and it prevents uninitialized memory warnings in Valgrind.
	   Even without erasing, the output would be valid (but nondeterministic).

	   Background information: The compressor stores short (at most 8 bytes)
	   substrings of the input already read in a hash table, and detects
	   repetitions by looking up such substrings in the hash table. If it
	   can find a substring, it checks whether the substring is really there
	   in the ring buffer (or it's just a hash collision). Should the hash
	   table become corrupt, this check makes sure that the output is
	   still valid, albeit the compression ratio would be bad.

	   The compressor populates the hash table from the ring buffer as it's
	   reading new bytes from the input. However, at the last few indexes of
	   the ring buffer, there are not enough bytes to build full-length
	   substrings from. Since the hash table always contains full-length
	   substrings, we erase with dummy zeros here to make sure that those
	   substrings will contain zeros at the end instead of uninitialized
	   data.

	   Please note that erasing is not necessary (because the
	   memory region is already initialized since he ring buffer
	   has a `tail' that holds a copy of the beginning,) so we
	   skip erasing if we have already gone around at least once in
	   the ring buffer.

	   Only clear during the first round of ring-buffer writes. On
	   subsequent rounds data in the ring-buffer would be affected. */
	if ringbuffer_.pos_ <= ringbuffer_.mask_ {
		/* This is the first time when the ring buffer is being written.
		   We clear 7 bytes just after the bytes that have been copied from
		   the input buffer.

		   The ring-buffer has a "tail" that holds a copy of the beginning,
		   but only once the ring buffer has been fully written once, i.e.,
		   pos <= mask. For the first time, we need to write values
		   in this tail (where index may be larger than mask), so that
		   we have exactly defined behavior and don't read uninitialized
		   memory. Due to performance reasons, hashing reads data using a
		   LOAD64, which can go 7 bytes beyond the bytes written in the
		   ring-buffer. */
		for i := 0; i < int(7); i++ {
			ringbuffer_.buffer_[ringbuffer_.pos_:][i] = 0
		}
	}
}

/*
Marks all input as processed.

	Returns true if position wrapping occurs.
*/
func updateLastProcessedPos(s *Writer) bool {
	var wrappedLastProcessedPos uint32 = wrapPosition(s.lastProcessedPos)
	var wrappedInputPos uint32 = wrapPosition(s.inputPos)
	s.lastProcessedPos = s.inputPos
	return wrappedInputPos < wrappedLastProcessedPos
}

func extendLastCommand(s *Writer, bytes *uint32, wrappedLastProcessedPos *uint32) {
	var lastCommand *command = &s.commands[len(s.commands)-1]
	var data []byte = s.ringbuffer_.buffer_
	var mask uint32 = s.ringbuffer_.mask_
	var maxBackwardDistance uint64 = ((uint64(1)) << s.params.lgwin) - windowGap
	var lastCopyLen uint64 = uint64(lastCommand.copy_len_) & 0x1FFFFFF
	var lastProcessedPos uint64 = s.lastProcessedPos - lastCopyLen
	var maxDistance uint64
	if lastProcessedPos < maxBackwardDistance {
		maxDistance = lastProcessedPos
	} else {
		maxDistance = maxBackwardDistance
	}
	var cmdDist uint64 = uint64(s.distCache[0])
	var distanceCode uint32 = commandRestoreDistanceCode(lastCommand, &s.params.dist)
	if distanceCode < numDistanceShortCodes || uint64(distanceCode-(numDistanceShortCodes-1)) == cmdDist {
		if cmdDist <= maxDistance {
			for *bytes != 0 && data[*wrappedLastProcessedPos&mask] == data[(uint64(*wrappedLastProcessedPos)-cmdDist)&uint64(mask)] {
				lastCommand.copy_len_++
				*bytes--
				*wrappedLastProcessedPos++
			}
		}

		/* The copy length is at most the metablock size, and thus expressible. */
		getLengthCode(uint(lastCommand.insert_len_), uint(int(lastCommand.copy_len_&0x1FFFFFF)+int(lastCommand.copy_len_>>25)), lastCommand.dist_prefix_&0x3FF == 0, &lastCommand.cmd_prefix_)
	}
}

/*
Processes the accumulated input data and writes
the new output meta-block to s.dest, if one has been
created (otherwise the processed input data is buffered internally).
If |is_last| or |force_flush| is true, an output meta-block is
always created. However, until |is_last| is true encoder may retain up
to 7 bits of the last byte of output. To force encoder to dump the remaining
bits use WriteMetadata() to append an empty meta-data block.
Returns false if the size of the input data is larger than
input_block_size().
*/
func encodeData(s *Writer, isLast bool, forceFlush bool) bool {
	var delta uint64 = unprocessedInputSize(s)
	var bytes uint32 = uint32(delta)
	var wrappedLastProcessedPos uint32 = wrapPosition(s.lastProcessedPos)
	var data []byte
	var mask uint32
	var literalContextMode int

	data = s.ringbuffer_.buffer_
	mask = s.ringbuffer_.mask_

	/* Adding more blocks after "last" block is forbidden. */
	if s.isLastBlockEmitted {
		return false
	}
	if isLast {
		s.isLastBlockEmitted = true
	}

	if delta > uint64(inputBlockSize(s)) {
		return false
	}

	if s.params.quality == fastTwoPassCompressionQuality {
		if s.commandBuf == nil || cap(s.commandBuf) < int(kCompressFragmentTwoPassBlockSize) {
			s.commandBuf = make([]uint32, kCompressFragmentTwoPassBlockSize)
			s.literalBuf = make([]byte, kCompressFragmentTwoPassBlockSize)
		} else {
			s.commandBuf = s.commandBuf[:kCompressFragmentTwoPassBlockSize]
			s.literalBuf = s.literalBuf[:kCompressFragmentTwoPassBlockSize]
		}
	}

	if s.params.quality == fastOnePassCompressionQuality || s.params.quality == fastTwoPassCompressionQuality {
		var storage []byte
		var storageIx uint = uint(s.lastBytesBits)
		var tableSize uint
		var table []int

		if delta == 0 && !isLast {
			/* We have no new input data and we don't have to finish the stream, so
			   nothing to do. */
			return true
		}

		storage = s.getStorage(int(2*bytes + 503))
		storage[0] = byte(s.lastBytes)
		storage[1] = byte(s.lastBytes >> 8)
		table = getHashTable(s, s.params.quality, uint(bytes), &tableSize)
		if s.params.quality == fastOnePassCompressionQuality {
			compressFragmentFast(data[wrappedLastProcessedPos&mask:], uint(bytes), isLast, table, tableSize, s.cmdDepths[:], s.cmdBits[:], &s.cmdCodeNumbits, s.cmdCode[:], &storageIx, storage)
		} else {
			compressFragmentTwoPass(data[wrappedLastProcessedPos&mask:], uint(bytes), isLast, s.commandBuf, s.literalBuf, table, tableSize, &storageIx, storage)
		}

		s.lastBytes = uint16(storage[storageIx>>3])
		s.lastBytesBits = byte(storageIx & 7)
		updateLastProcessedPos(s)
		s.writeOutput(storage[:storageIx>>3])
		return true
	}
	{
		/* Theoretical max number of commands is 1 per 2 bytes. */
		newsize := len(s.commands) + int(bytes)/2 + 1
		if newsize > cap(s.commands) {
			/* Reserve a bit more memory to allow merging with a next block
			   without reallocation: that would impact speed. */
			newsize += int(bytes/4) + 16

			newCommands := make([]command, len(s.commands), newsize)
			if s.commands != nil {
				copy(newCommands, s.commands)
			}

			s.commands = newCommands
		}
	}

	initOrStitchToPreviousBlock(&s.hasher_, data, uint(mask), &s.params, uint(wrappedLastProcessedPos), uint(bytes), isLast)

	literalContextMode = chooseContextMode(&s.params, data, uint(wrapPosition(s.lastFlushPos)), uint(mask), uint(s.inputPos-s.lastFlushPos))

	if len(s.commands) != 0 && s.lastInsertLen == 0 {
		extendLastCommand(s, &bytes, &wrappedLastProcessedPos)
	}

	if s.params.quality == zopflificationQuality {
		assert(s.params.hasher.type_ == 10)
		createZopfliBackwardReferences(uint(bytes), uint(wrappedLastProcessedPos), data, uint(mask), &s.params, s.hasher_.(*h10), s.distCache[:], &s.lastInsertLen, &s.commands, &s.numLiterals)
	} else if s.params.quality == hqZopflificationQuality {
		assert(s.params.hasher.type_ == 10)
		createHqZopfliBackwardReferences(uint(bytes), uint(wrappedLastProcessedPos), data, uint(mask), &s.params, s.hasher_, s.distCache[:], &s.lastInsertLen, &s.commands, &s.numLiterals)
	} else {
		createBackwardReferences(uint(bytes), uint(wrappedLastProcessedPos), data, uint(mask), &s.params, s.hasher_, s.distCache[:], &s.lastInsertLen, &s.commands, &s.numLiterals)
	}
	{
		var maxLength uint = maxMetablockSize(&s.params)
		var maxLiterals uint = maxLength / 8
		maxCommands := int(maxLength / 8)
		var processedBytes uint = uint(s.inputPos - s.lastFlushPos)
		var nextInputFitsMetablock bool = processedBytes+inputBlockSize(s) <= maxLength
		var shouldFlush bool = s.params.quality < minQualityForBlockSplit && s.numLiterals+uint(len(s.commands)) >= maxNumDelayedSymbols
		/* If maximal possible additional block doesn't fit metablock, flush now. */
		/* TODO: Postpone decision until next block arrives? */

		/* If block splitting is not used, then flush as soon as there is some
		   amount of commands / literals produced. */
		if !isLast && !forceFlush && !shouldFlush && nextInputFitsMetablock && s.numLiterals < maxLiterals && len(s.commands) < maxCommands {
			/* Merge with next input block. Everything will happen later. */
			if updateLastProcessedPos(s) {
				hasherReset(s.hasher_)
			}

			return true
		}
	}

	/* Create the last insert-only command. */
	if s.lastInsertLen > 0 {
		s.commands = append(s.commands, makeInsertCommand(s.lastInsertLen))
		s.numLiterals += s.lastInsertLen
		s.lastInsertLen = 0
	}

	if !isLast && s.inputPos == s.lastFlushPos {
		/* We have no new input data and we don't have to finish the stream, so
		   nothing to do. */
		return true
	}

	assert(s.inputPos >= s.lastFlushPos)
	assert(s.inputPos > s.lastFlushPos || isLast)
	assert(s.inputPos-s.lastFlushPos <= 1<<24)
	{
		var metablockSize uint32 = uint32(s.inputPos - s.lastFlushPos)
		var storage []byte = s.getStorage(int(2*metablockSize + 503))
		var storageIx uint = uint(s.lastBytesBits)
		storage[0] = byte(s.lastBytes)
		storage[1] = byte(s.lastBytes >> 8)
		writeMetaBlockInternal(data, uint(mask), s.lastFlushPos, uint(metablockSize), isLast, literalContextMode, &s.params, s.prevByte, s.prevByte2, s.numLiterals, s.commands, s.savedDistCache[:], s.distCache[:], &storageIx, storage)
		s.lastBytes = uint16(storage[storageIx>>3])
		s.lastBytesBits = byte(storageIx & 7)
		s.lastFlushPos = s.inputPos
		if updateLastProcessedPos(s) {
			hasherReset(s.hasher_)
		}

		if s.lastFlushPos > 0 {
			s.prevByte = data[(uint32(s.lastFlushPos)-1)&mask]
		}

		if s.lastFlushPos > 1 {
			s.prevByte2 = data[uint32(s.lastFlushPos-2)&mask]
		}

		s.commands = s.commands[:0]
		s.numLiterals = 0

		/* Save the state of the distance cache in case we need to restore it for
		   emitting an uncompressed block. */
		copy(s.savedDistCache[:], s.distCache[:])

		s.writeOutput(storage[:storageIx>>3])
		return true
	}
}

/*
Dumps remaining output bits and metadata header to |header|.

	Returns number of produced bytes.
	REQUIRED: |header| should be 8-byte aligned and at least 16 bytes long.
	REQUIRED: |block_size| <= (1 << 24).
*/
func writeMetadataHeader(s *Writer, blockSize uint, header []byte) uint {
	storageIx := uint(s.lastBytesBits)
	header[0] = byte(s.lastBytes)
	header[1] = byte(s.lastBytes >> 8)
	s.lastBytes = 0
	s.lastBytesBits = 0

	writeBits(1, 0, &storageIx, header)
	writeBits(2, 3, &storageIx, header)
	writeBits(1, 0, &storageIx, header)
	if blockSize == 0 {
		writeBits(2, 0, &storageIx, header)
	} else {
		var nbits uint32
		if blockSize == 1 {
			nbits = 0
		} else {
			nbits = log2FloorNonZero(uint(uint32(blockSize)-1)) + 1
		}
		var nbytes uint32 = (nbits + 7) / 8
		writeBits(2, uint64(nbytes), &storageIx, header)
		writeBits(uint(8*nbytes), uint64(blockSize)-1, &storageIx, header)
	}

	return (storageIx + 7) >> 3
}

func injectBytePaddingBlock(s *Writer) {
	var seal uint32 = uint32(s.lastBytes)
	var sealBits uint = uint(s.lastBytesBits)
	s.lastBytes = 0
	s.lastBytesBits = 0

	/* is_last = 0, data_nibbles = 11, reserved = 0, meta_nibbles = 00 */
	seal |= 0x6 << sealBits

	sealBits += 6

	destination := s.tinyBuf.u8[:]

	destination[0] = byte(seal)
	if sealBits > 8 {
		destination[1] = byte(seal >> 8)
	}
	if sealBits > 16 {
		destination[2] = byte(seal >> 16)
	}
	s.writeOutput(destination[:(sealBits+7)>>3])
}

func checkFlushComplete(s *Writer) {
	if s.streamState == streamFlushRequested && s.err == nil {
		s.streamState = streamProcessing
	}
}

func encoderCompressStreamFast(s *Writer, op int, availableIn *uint, nextIn *[]byte) bool {
	var blockSizeLimit uint = uint(1) << s.params.lgwin
	var bufSize uint = brotliMinSizeT(kCompressFragmentTwoPassBlockSize, brotliMinSizeT(*availableIn, blockSizeLimit))
	var commandBuf []uint32 = nil
	var literalBuf []byte = nil
	if s.params.quality != fastOnePassCompressionQuality && s.params.quality != fastTwoPassCompressionQuality {
		return false
	}

	if s.params.quality == fastTwoPassCompressionQuality {
		if s.commandBuf == nil || cap(s.commandBuf) < int(bufSize) {
			s.commandBuf = make([]uint32, bufSize)
			s.literalBuf = make([]byte, bufSize)
		} else {
			s.commandBuf = s.commandBuf[:bufSize]
			s.literalBuf = s.literalBuf[:bufSize]
		}

		commandBuf = s.commandBuf
		literalBuf = s.literalBuf
	}

	for {
		if s.streamState == streamFlushRequested && s.lastBytesBits != 0 {
			injectBytePaddingBlock(s)
			continue
		}

		/* Compress block only when stream is not
		   finished, there is no pending flush request, and there is either
		   additional input or pending operation. */
		if s.streamState == streamProcessing && (*availableIn != 0 || op != int(operationProcess)) {
			var blockSize uint = brotliMinSizeT(blockSizeLimit, *availableIn)
			var isLast bool = (*availableIn == blockSize) && (op == int(operationFinish))
			var forceFlush bool = (*availableIn == blockSize) && (op == int(operationFlush))
			var maxOutSize uint = 2*blockSize + 503
			var storage []byte = nil
			var storageIx uint = uint(s.lastBytesBits)
			var tableSize uint
			var table []int

			if forceFlush && blockSize == 0 {
				s.streamState = streamFlushRequested
				continue
			}

			storage = s.getStorage(int(maxOutSize))

			storage[0] = byte(s.lastBytes)
			storage[1] = byte(s.lastBytes >> 8)
			table = getHashTable(s, s.params.quality, blockSize, &tableSize)

			if s.params.quality == fastOnePassCompressionQuality {
				compressFragmentFast(*nextIn, blockSize, isLast, table, tableSize, s.cmdDepths[:], s.cmdBits[:], &s.cmdCodeNumbits, s.cmdCode[:], &storageIx, storage)
			} else {
				compressFragmentTwoPass(*nextIn, blockSize, isLast, commandBuf, literalBuf, table, tableSize, &storageIx, storage)
			}

			*nextIn = (*nextIn)[blockSize:]
			*availableIn -= blockSize
			var outBytes uint = storageIx >> 3
			s.writeOutput(storage[:outBytes])

			s.lastBytes = uint16(storage[storageIx>>3])
			s.lastBytesBits = byte(storageIx & 7)

			if forceFlush {
				s.streamState = streamFlushRequested
			}
			if isLast {
				s.streamState = streamFinished
			}
			continue
		}

		break
	}

	checkFlushComplete(s)
	return true
}

func processMetadata(s *Writer, availableIn *uint, nextIn *[]byte) bool {
	if *availableIn > 1<<24 {
		return false
	}

	/* Switch to metadata block workflow, if required. */
	if s.streamState == streamProcessing {
		s.remainingMetadataBytes = uint32(*availableIn)
		s.streamState = streamMetadataHead
	}

	if s.streamState != streamMetadataHead && s.streamState != streamMetadataBody {
		return false
	}

	for {
		if s.streamState == streamFlushRequested && s.lastBytesBits != 0 {
			injectBytePaddingBlock(s)
			continue
		}

		if s.inputPos != s.lastFlushPos {
			var result bool = encodeData(s, false, true)
			if !result {
				return false
			}
			continue
		}

		if s.streamState == streamMetadataHead {
			n := writeMetadataHeader(s, uint(s.remainingMetadataBytes), s.tinyBuf.u8[:])
			s.writeOutput(s.tinyBuf.u8[:n])
			s.streamState = streamMetadataBody
			continue
		} else {
			/* Exit workflow only when there is no more input and no more output.
			   Otherwise client may continue producing empty metadata blocks. */
			if s.remainingMetadataBytes == 0 {
				s.remainingMetadataBytes = math.MaxUint32
				s.streamState = streamProcessing
				break
			}

			/* This guarantees progress in "TakeOutput" workflow. */
			var c uint32 = brotliMinUint32T(s.remainingMetadataBytes, 16)
			copy(s.tinyBuf.u8[:], (*nextIn)[:c])
			*nextIn = (*nextIn)[c:]
			*availableIn -= uint(c)
			s.remainingMetadataBytes -= c
			s.writeOutput(s.tinyBuf.u8[:c])

			continue
		}
	}

	return true
}

func updateSizeHint(s *Writer, availableIn uint) {
	if s.params.sizeHint == 0 {
		var delta uint64 = unprocessedInputSize(s)
		var tail uint64 = uint64(availableIn)
		var limit uint32 = 1 << 30
		var total uint32
		if (delta >= uint64(limit)) || (tail >= uint64(limit)) || ((delta + tail) >= uint64(limit)) {
			total = limit
		} else {
			total = uint32(delta + tail)
		}

		s.params.sizeHint = uint(total)
	}
}

func encoderCompressStream(s *Writer, op int, availableIn *uint, nextIn *[]byte) bool {
	if !ensureInitialized(s) {
		return false
	}

	/* Unfinished metadata block; check requirements. */
	if s.remainingMetadataBytes != math.MaxUint32 {
		if uint32(*availableIn) != s.remainingMetadataBytes {
			return false
		}
		if op != int(operationEmitMetadata) {
			return false
		}
	}

	if op == int(operationEmitMetadata) {
		updateSizeHint(s, 0) /* First data metablock might be emitted here. */
		return processMetadata(s, availableIn, nextIn)
	}

	if s.streamState == streamMetadataHead || s.streamState == streamMetadataBody {
		return false
	}

	if s.streamState != streamProcessing && *availableIn != 0 {
		return false
	}

	if s.params.quality == fastOnePassCompressionQuality || s.params.quality == fastTwoPassCompressionQuality {
		return encoderCompressStreamFast(s, op, availableIn, nextIn)
	}

	for {
		var remainingBlockSize uint = remainingInputBlockSize(s)

		if remainingBlockSize != 0 && *availableIn != 0 {
			var copyInputSize uint = brotliMinSizeT(remainingBlockSize, *availableIn)
			copyInputToRingBuffer(s, copyInputSize, *nextIn)
			*nextIn = (*nextIn)[copyInputSize:]
			*availableIn -= copyInputSize
			continue
		}

		if s.streamState == streamFlushRequested && s.lastBytesBits != 0 {
			injectBytePaddingBlock(s)
			continue
		}

		/* Compress data only when stream is not
		   finished and there is no pending flush request. */
		if s.streamState == streamProcessing {
			if remainingBlockSize == 0 || op != int(operationProcess) {
				var isLast bool = (*availableIn == 0) && op == int(operationFinish)
				var forceFlush bool = (*availableIn == 0) && op == int(operationFlush)
				var result bool
				updateSizeHint(s, *availableIn)
				result = encodeData(s, isLast, forceFlush)
				if !result {
					return false
				}
				if forceFlush {
					s.streamState = streamFlushRequested
				}
				if isLast {
					s.streamState = streamFinished
				}
				continue
			}
		}

		break
	}

	checkFlushComplete(s)
	return true
}

func (w *Writer) writeOutput(data []byte) {
	if w.err != nil {
		return
	}

	_, w.err = w.dst.Write(data)
	if w.err == nil {
		checkFlushComplete(w)
	}
}
