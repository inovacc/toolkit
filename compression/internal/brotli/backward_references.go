package brotli

import (
	"sync"
)

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Function to find backward reference copies. */

func computeDistanceCode(distance uint, maxDistance uint, distCache []int) uint {
	if distance <= maxDistance {
		var distancePlus3 uint = distance + 3
		var offset0 uint = distancePlus3 - uint(distCache[0])
		var offset1 uint = distancePlus3 - uint(distCache[1])
		if distance == uint(distCache[0]) {
			return 0
		} else if distance == uint(distCache[1]) {
			return 1
		} else if offset0 < 7 {
			return (0x9750468 >> (4 * offset0)) & 0xF
		} else if offset1 < 7 {
			return (0xFDB1ACE >> (4 * offset1)) & 0xF
		} else if distance == uint(distCache[2]) {
			return 2
		} else if distance == uint(distCache[3]) {
			return 3
		}
	}

	return distance + numDistanceShortCodes - 1
}

var hasherSearchResultPool sync.Pool

func createBackwardReferences(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint, params *encoderParams, hasher hasherHandle, distCache []int, lastInsertLen *uint, commands *[]command, numLiterals *uint) {
	var maxBackwardLimit uint = maxBackwardLimitFn(params.lgwin)
	var insertLength uint = *lastInsertLen
	var posEnd uint = position + numBytes
	var storeEnd uint
	if numBytes >= hasher.StoreLookahead() {
		storeEnd = position + numBytes - hasher.StoreLookahead() + 1
	} else {
		storeEnd = position
	}
	var randomHeuristicsWindowSize uint = literalSpreeLengthForSparseSearch(params)
	var applyRandomHeuristics uint = position + randomHeuristicsWindowSize
	var gap uint = 0
	/* Set maximum distance, see section 9.1. of the spec. */

	const kMinScore uint = scoreBase + 100

	/* For speed-up heuristics for random data. */

	/* Minimum score to accept a backward reference. */
	hasher.PrepareDistanceCache(distCache)
	sr2, _ := hasherSearchResultPool.Get().(*hasherSearchResult)
	if sr2 == nil {
		sr2 = &hasherSearchResult{}
	}
	sr, _ := hasherSearchResultPool.Get().(*hasherSearchResult)
	if sr == nil {
		sr = &hasherSearchResult{}
	}

	for position+hasher.HashTypeLength() < posEnd {
		var maxLength uint = posEnd - position
		var maxDistance uint = brotliMinSizeT(position, maxBackwardLimit)
		sr.len = 0
		sr.len_code_delta = 0
		sr.distance = 0
		sr.score = kMinScore
		hasher.FindLongestMatch(&params.dictionary, ringbuffer, ringbufferMask, distCache, position, maxLength, maxDistance, gap, params.dist.maxDistance, sr)
		if sr.score > kMinScore {
			/* Found a match. Let's look for something even better ahead. */
			var delayedBackwardReferencesInRow int = 0
			maxLength--
			for ; ; maxLength-- {
				var costDiffLazy uint = 175
				if params.quality < minQualityForExtensiveReferenceSearch {
					sr2.len = brotliMinSizeT(sr.len-1, maxLength)
				} else {
					sr2.len = 0
				}
				sr2.len_code_delta = 0
				sr2.distance = 0
				sr2.score = kMinScore
				maxDistance = brotliMinSizeT(position+1, maxBackwardLimit)
				hasher.FindLongestMatch(&params.dictionary, ringbuffer, ringbufferMask, distCache, position+1, maxLength, maxDistance, gap, params.dist.maxDistance, sr2)
				if sr2.score >= sr.score+costDiffLazy {
					/* Ok, let's just write one byte for now and start a match from the
					   next byte. */
					position++

					insertLength++
					*sr = *sr2
					delayedBackwardReferencesInRow++
					if delayedBackwardReferencesInRow < 4 && position+hasher.HashTypeLength() < posEnd {
						continue
					}
				}

				break
			}

			applyRandomHeuristics = position + 2*sr.len + randomHeuristicsWindowSize
			maxDistance = brotliMinSizeT(position, maxBackwardLimit)
			{
				/* The first 16 codes are special short-codes,
				   and the minimum offset is 1. */
				var distanceCodeVar uint = computeDistanceCode(sr.distance, maxDistance+gap, distCache)
				if (sr.distance <= (maxDistance + gap)) && distanceCodeVar > 0 {
					distCache[3] = distCache[2]
					distCache[2] = distCache[1]
					distCache[1] = distCache[0]
					distCache[0] = int(sr.distance)
					hasher.PrepareDistanceCache(distCache)
				}

				*commands = append(*commands, makeCommand(&params.dist, insertLength, sr.len, sr.len_code_delta, distanceCodeVar))
			}

			*numLiterals += insertLength
			insertLength = 0
			/* Put the hash keys into the table, if there are enough bytes left.
			   Depending on the hasher implementation, it can push all positions
			   in the given range or only a subset of them.
			   Avoid hash poisoning with RLE data. */
			{
				var rangeStart uint = position + 2
				var rangeEnd uint = brotliMinSizeT(position+sr.len, storeEnd)
				if sr.distance < sr.len>>2 {
					rangeStart = brotliMinSizeT(rangeEnd, brotliMaxSizeT(rangeStart, position+sr.len-(sr.distance<<2)))
				}

				hasher.StoreRange(ringbuffer, ringbufferMask, rangeStart, rangeEnd)
			}

			position += sr.len
		} else {
			insertLength++
			position++

			/* If we have not seen matches for a long time, we can skip some
			   match lookups. Unsuccessful match lookups are very very expensive
			   and this kind of a heuristic speeds up compression quite
			   a lot. */
			if position > applyRandomHeuristics {
				/* Going through uncompressible data, jump. */
				if position > applyRandomHeuristics+4*randomHeuristicsWindowSize {
					var kMargin uint = brotliMaxSizeT(hasher.StoreLookahead()-1, 4)
					/* It is quite a long time since we saw a copy, so we assume
					   that this data is not compressible, and store hashes less
					   often. Hashes of non compressible data are less likely to
					   turn out to be useful in the future, too, so we store less of
					   them to not to flood out the hash table of good compressible
					   data. */

					var posJump uint = brotliMinSizeT(position+16, posEnd-kMargin)
					for ; position < posJump; position += 4 {
						hasher.Store(ringbuffer, ringbufferMask, position)
						insertLength += 4
					}
				} else {
					var kMargin uint = brotliMaxSizeT(hasher.StoreLookahead()-1, 2)
					var posJump uint = brotliMinSizeT(position+8, posEnd-kMargin)
					for ; position < posJump; position += 2 {
						hasher.Store(ringbuffer, ringbufferMask, position)
						insertLength += 2
					}
				}
			}
		}
	}

	insertLength += posEnd - position
	*lastInsertLen = insertLength

	hasherSearchResultPool.Put(sr)
	hasherSearchResultPool.Put(sr2)
}
