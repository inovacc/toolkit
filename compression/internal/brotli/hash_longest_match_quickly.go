package brotli

import "encoding/binary"

/* Copyright 2010 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* For BUCKET_SWEEP == 1, enabling the dictionary lookup makes compression
   a little faster (0.5% - 1%) and it compresses 0.15% better on small text
   and HTML inputs. */

func (*hashLongestMatchQuickly) HashTypeLength() uint {
	return 8
}

func (*hashLongestMatchQuickly) StoreLookahead() uint {
	return 8
}

/*
HashBytes is the function that chooses the bucket to place

	the address in. The HashLongestMatch and hashLongestMatchQuickly
	classes have separate, different implementations of hashing.
*/
func (h *hashLongestMatchQuickly) HashBytes(data []byte) uint32 {
	var hash uint64 = (binary.LittleEndian.Uint64(data) << (64 - 8*h.hashLen)) * kHashMul64

	/* The higher bits contain more mixture from the multiplication,
	   so we take our results from there. */
	return uint32(hash >> (64 - h.bucketBits))
}

/*
A (forgetful) hash table to the data seen by the compressor, to

	help create backward references to previous data.

	This is a hash map of fixed size (1 << 16). Starting from the
	given index, 1 buckets are used to store values of a key.
*/
type hashLongestMatchQuickly struct {
	hasherCommon

	bucketBits    uint
	bucketSweep   int
	hashLen       uint
	useDictionary bool

	buckets []uint32
}

func (h *hashLongestMatchQuickly) Initialize(params *encoderParams) {
	h.buckets = make([]uint32, 1<<h.bucketBits+h.bucketSweep)
}

func (h *hashLongestMatchQuickly) Prepare(oneShot bool, inputSize uint, data []byte) {
	var partialPrepareThreshold uint = (4 << h.bucketBits) >> 7
	/* Partial preparation is 100 times slower (per socket). */
	if oneShot && inputSize <= partialPrepareThreshold {
		var i uint
		for i = 0; i < inputSize; i++ {
			var key uint32 = h.HashBytes(data[i:])
			for j := 0; j < h.bucketSweep; j++ {
				h.buckets[key+uint32(j)] = 0
			}
		}
	} else {
		/* It is not strictly necessary to fill this buffer here, but
		   not filling will make the results of the compression stochastic
		   (but correct). This is because random data would cause the
		   system to find accidentally good backward references here and there. */
		for i := range h.buckets {
			h.buckets[i] = 0
		}
	}
}

/*
Look at 5 bytes at &data[ix & mask].

	Compute a hash from these, and store the value somewhere within
	[ix .. ix+3].
*/
func (h *hashLongestMatchQuickly) Store(data []byte, mask uint, ix uint) {
	var key uint32 = h.HashBytes(data[ix&mask:])
	var off uint32 = uint32(ix>>3) % uint32(h.bucketSweep)
	/* Wiggle the value with the bucket sweep range. */
	h.buckets[key+off] = uint32(ix)
}

func (h *hashLongestMatchQuickly) StoreRange(data []byte, mask uint, ix_start uint, ix_end uint) {
	var i uint
	for i = ix_start; i < ix_end; i++ {
		h.Store(data, mask, i)
	}
}

func (h *hashLongestMatchQuickly) StitchToPreviousBlock(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint) {
	if numBytes >= h.HashTypeLength()-1 && position >= 3 {
		/* Prepare the hashes for three last bytes of the last write.
		   These could not be calculated before, since they require knowledge
		   of both the previous and the current block. */
		h.Store(ringbuffer, ringbufferMask, position-3)
		h.Store(ringbuffer, ringbufferMask, position-2)
		h.Store(ringbuffer, ringbufferMask, position-1)
	}
}

func (*hashLongestMatchQuickly) PrepareDistanceCache(distance_cache []int) {
}

/*
Find a longest backward match of &data[cur_ix & ring_buffer_mask]

	up to the length of max_length and stores the position cur_ix in the
	hash table.

	Does not look for matches longer than max_length.
	Does not look for matches further away than max_backward.
	Writes the best match into |out|.
	|out|->score is updated only if a better match is found.
*/
func (h *hashLongestMatchQuickly) FindLongestMatch(dictionary *encoderDictionary, data []byte, ringBufferMask uint, distanceCache []int, curIx uint, maxLength uint, maxBackward uint, gap uint, maxDistance uint, out *hasherSearchResult) {
	var bestLenIn uint = out.len
	var curIxMasked uint = curIx & ringBufferMask
	var key uint32 = h.HashBytes(data[curIxMasked:])
	var compareChar int = int(data[curIxMasked+bestLenIn])
	var minScore uint = out.score
	var bestScore uint = out.score
	var bestLen uint = bestLenIn
	var cachedBackward uint = uint(distanceCache[0])
	var prevIx uint = curIx - cachedBackward
	var bucket []uint32
	out.len_code_delta = 0
	if prevIx < curIx {
		prevIx &= uint(uint32(ringBufferMask))
		if compareChar == int(data[prevIx+bestLen]) {
			var limit uint = findMatchLengthWithLimit(data[prevIx:], data[curIxMasked:], maxLength)
			if limit >= 4 {
				var score uint = backwardReferenceScoreUsingLastDistance(uint(limit))
				if bestScore < score {
					bestScore = score
					bestLen = uint(limit)
					out.len = uint(limit)
					out.distance = cachedBackward
					out.score = bestScore
					compareChar = int(data[curIxMasked+bestLen])
					if h.bucketSweep == 1 {
						h.buckets[key] = uint32(curIx)
						return
					}
				}
			}
		}
	}

	if h.bucketSweep == 1 {
		var backward uint
		var l uint

		/* Only one to look for, don't bother to prepare for a loop. */
		prevIx = uint(h.buckets[key])

		h.buckets[key] = uint32(curIx)
		backward = curIx - prevIx
		prevIx &= uint(uint32(ringBufferMask))
		if compareChar != int(data[prevIx+bestLenIn]) {
			return
		}

		if backward == 0 || backward > maxBackward {
			return
		}

		l = findMatchLengthWithLimit(data[prevIx:], data[curIxMasked:], maxLength)
		if l >= 4 {
			var score uint = backwardReferenceScore(uint(l), backward)
			if bestScore < score {
				out.len = uint(l)
				out.distance = backward
				out.score = score
				return
			}
		}
	} else {
		bucket = h.buckets[key:]
		var i int
		prevIx = uint(bucket[0])
		bucket = bucket[1:]
		for i = 0; i < h.bucketSweep; (func() { i++; tmp3 := bucket; bucket = bucket[1:]; prevIx = uint(tmp3[0]) })() {
			var backward uint = curIx - prevIx
			var l uint
			prevIx &= uint(uint32(ringBufferMask))
			if compareChar != int(data[prevIx+bestLen]) {
				continue
			}

			if backward == 0 || backward > maxBackward {
				continue
			}

			l = findMatchLengthWithLimit(data[prevIx:], data[curIxMasked:], maxLength)
			if l >= 4 {
				var score uint = backwardReferenceScore(uint(l), backward)
				if bestScore < score {
					bestScore = score
					bestLen = uint(l)
					out.len = bestLen
					out.distance = backward
					out.score = score
					compareChar = int(data[curIxMasked+bestLen])
				}
			}
		}
	}

	if h.useDictionary && minScore == out.score {
		searchInStaticDictionary(dictionary, h, data[curIxMasked:], maxLength, maxBackward+gap, maxDistance, out, true)
	}

	h.buckets[key+uint32((curIx>>3)%uint(h.bucketSweep))] = uint32(curIx)
}
