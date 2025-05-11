package brotli

import "encoding/binary"

/* Copyright 2010 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/*
A (forgetful) hash table to the data seen by the compressor, to

	help create backward references to previous data.

	This is a hash map of fixed size (bucket_size_) to a ring buffer of
	fixed size (block_size_). The ring buffer contains the last block_size_
	index positions of the given hash key in the compressed data.
*/
func (*h5) HashTypeLength() uint {
	return 4
}

func (*h5) StoreLookahead() uint {
	return 4
}

/* HashBytes is the function that chooses the bucket to place the address in. */
func hashBytesH5(data []byte, shift int) uint32 {
	var h uint32 = binary.LittleEndian.Uint32(data) * kHashMul32

	/* The higher bits contain more mixture from the multiplication,
	   so we take our results from there. */
	return uint32(h >> uint(shift))
}

type h5 struct {
	hasherCommon
	bucketSize uint
	blockSize  uint
	hashShift  int
	blockMask  uint32
	num        []uint16
	buckets    []uint32
}

func (h *h5) Initialize(params *encoderParams) {
	h.hashShift = 32 - h.params.bucketBits
	h.bucketSize = uint(1) << uint(h.params.bucketBits)
	h.blockSize = uint(1) << uint(h.params.blockBits)
	h.blockMask = uint32(h.blockSize - 1)
	h.num = make([]uint16, h.bucketSize)
	h.buckets = make([]uint32, h.blockSize*h.bucketSize)
}

func (h *h5) Prepare(oneShot bool, inputSize uint, data []byte) {
	var num []uint16 = h.num
	var partialPrepareThreshold uint = h.bucketSize >> 6
	/* Partial preparation is 100 times slower (per socket). */
	if oneShot && inputSize <= partialPrepareThreshold {
		var i uint
		for i = 0; i < inputSize; i++ {
			var key uint32 = hashBytesH5(data[i:], h.hashShift)
			num[key] = 0
		}
	} else {
		for i := 0; i < int(h.bucketSize); i++ {
			num[i] = 0
		}
	}
}

/*
Look at 4 bytes at &data[ix & mask].

	Compute a hash from these, and store the value of ix at that position.
*/
func (h *h5) Store(data []byte, mask uint, ix uint) {
	var num []uint16 = h.num
	var key uint32 = hashBytesH5(data[ix&mask:], h.hashShift)
	var minorIx uint = uint(num[key]) & uint(h.blockMask)
	var offset uint = minorIx + uint(key<<uint(h.params.blockBits))
	h.buckets[offset] = uint32(ix)
	num[key]++
}

func (h *h5) StoreRange(data []byte, mask uint, ixStart uint, ixEnd uint) {
	var i uint
	for i = ixStart; i < ixEnd; i++ {
		h.Store(data, mask, i)
	}
}

func (h *h5) StitchToPreviousBlock(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint) {
	if numBytes >= h.HashTypeLength()-1 && position >= 3 {
		/* Prepare the hashes for three last bytes of the last write.
		   These could not be calculated before, since they require knowledge
		   of both the previous and the current block. */
		h.Store(ringbuffer, ringbufferMask, position-3)
		h.Store(ringbuffer, ringbufferMask, position-2)
		h.Store(ringbuffer, ringbufferMask, position-1)
	}
}

func (h *h5) PrepareDistanceCache(distanceCache []int) {
	prepareDistanceCache(distanceCache, h.params.numLastDistancesToCheck)
}

/*
Find a longest backward match of &data[cur_ix] up to the length of

	max_length and stores the position cur_ix in the hash table.

	REQUIRES: PrepareDistanceCacheH5 must be invoked for current distance cache
	          values; if this method is invoked repeatedly with the same distance
	          cache values, it is enough to invoke PrepareDistanceCacheH5 once.

	Does not look for matches longer than max_length.
	Does not look for matches further away than max_backward.
	Writes the best match into |out|.
	|out|->score is updated only if a better match is found.
*/
func (h *h5) FindLongestMatch(dictionary *encoderDictionary, data []byte, ringBufferMask uint, distanceCache []int, curIx uint, maxLength uint, maxBackward uint, gap uint, maxDistance uint, out *hasherSearchResult) {
	var num []uint16 = h.num
	var buckets []uint32 = h.buckets
	var curIxMasked uint = curIx & ringBufferMask
	var minScore uint = out.score
	var bestScore uint = out.score
	var bestLen uint = out.len
	var i uint
	var bucket []uint32
	/* Don't accept a short copy from far away. */
	out.len = 0

	out.len_code_delta = 0

	/* Try last distance first. */
	for i = 0; i < uint(h.params.numLastDistancesToCheck); i++ {
		var backward uint = uint(distanceCache[i])
		var prevIx uint = uint(curIx - backward)
		if prevIx >= curIx {
			continue
		}

		if backward > maxBackward {
			continue
		}

		prevIx &= ringBufferMask

		if curIxMasked+bestLen > ringBufferMask || prevIx+bestLen > ringBufferMask || data[curIxMasked+bestLen] != data[prevIx+bestLen] {
			continue
		}
		{
			var limit uint = findMatchLengthWithLimit(data[prevIx:], data[curIxMasked:], maxLength)
			if limit >= 3 || (limit == 2 && i < 2) {
				/* Comparing for >= 2 does not change the semantics, but just saves for
				   a few unnecessary binary logarithms in backward reference score,
				   since we are not interested in such short matches. */
				var score uint = backwardReferenceScoreUsingLastDistance(uint(limit))
				if bestScore < score {
					if i != 0 {
						score -= backwardReferencePenaltyUsingLastDistance(i)
					}
					if bestScore < score {
						bestScore = score
						bestLen = uint(limit)
						out.len = bestLen
						out.distance = backward
						out.score = bestScore
					}
				}
			}
		}
	}
	{
		var key uint32 = hashBytesH5(data[curIxMasked:], h.hashShift)
		bucket = buckets[key<<uint(h.params.blockBits):]
		var down uint
		if uint(num[key]) > h.blockSize {
			down = uint(num[key]) - h.blockSize
		} else {
			down = 0
		}
		for i = uint(num[key]); i > down; {
			var prevIx uint
			i--
			prevIx = uint(bucket[uint32(i)&h.blockMask])
			var backward uint = curIx - prevIx
			if backward > maxBackward {
				break
			}

			prevIx &= ringBufferMask
			if curIxMasked+bestLen > ringBufferMask || prevIx+bestLen > ringBufferMask || data[curIxMasked+bestLen] != data[prevIx+bestLen] {
				continue
			}
			{
				var limit uint = findMatchLengthWithLimit(data[prevIx:], data[curIxMasked:], maxLength)
				if limit >= 4 {
					/* Comparing for >= 3 does not change the semantics, but just saves
					   for a few unnecessary binary logarithms in backward reference
					   score, since we are not interested in such short matches. */
					var score uint = backwardReferenceScore(uint(limit), backward)
					if bestScore < score {
						bestScore = score
						bestLen = uint(limit)
						out.len = bestLen
						out.distance = backward
						out.score = bestScore
					}
				}
			}
		}

		bucket[uint32(num[key])&h.blockMask] = uint32(curIx)
		num[key]++
	}

	if minScore == out.score {
		searchInStaticDictionary(dictionary, h, data[curIxMasked:], maxLength, maxBackward+gap, maxDistance, out, false)
	}
}
