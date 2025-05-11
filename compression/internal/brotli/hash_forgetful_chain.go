package brotli

import "encoding/binary"

/* Copyright 2016 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

func (*hashForgetfulChain) HashTypeLength() uint {
	return 4
}

func (*hashForgetfulChain) StoreLookahead() uint {
	return 4
}

/* HashBytes is the function that chooses the bucket to place the address in.*/
func (h *hashForgetfulChain) HashBytes(data []byte) uint {
	var hash uint32 = binary.LittleEndian.Uint32(data) * kHashMul32

	/* The higher bits contain more mixture from the multiplication,
	   so we take our results from there. */
	return uint(hash >> (32 - h.bucketBits))
}

type slot struct {
	delta uint16
	next  uint16
}

/*
A (forgetful) hash table to the data seen by the compressor, to

	help create backward references to previous data.

	Hashes are stored in chains which are bucketed to groups. Group of chains
	share a storage "bank". When more than "bank size" chain nodes are added,
	oldest nodes are replaced; this way several chains may share a tail.
*/
type hashForgetfulChain struct {
	hasherCommon

	bucketBits              uint
	numBanks                uint
	bankBits                uint
	numLastDistancesToCheck int

	addr        []uint32
	head        []uint16
	tinyHash    [65536]byte
	banks       [][]slot
	freeSlotIdx []uint16
	maxHops     uint
}

func (h *hashForgetfulChain) Initialize(params *encoderParams) {
	var q uint
	if params.quality > 6 {
		q = 7
	} else {
		q = 8
	}
	h.maxHops = q << uint(params.quality-4)

	bankSize := 1 << h.bankBits
	bucketSize := 1 << h.bucketBits

	h.addr = make([]uint32, bucketSize)
	h.head = make([]uint16, bucketSize)
	h.banks = make([][]slot, h.numBanks)
	for i := range h.banks {
		h.banks[i] = make([]slot, bankSize)
	}
	h.freeSlotIdx = make([]uint16, h.numBanks)
}

func (h *hashForgetfulChain) Prepare(oneShot bool, inputSize uint, data []byte) {
	var partialPrepareThreshold uint = (1 << h.bucketBits) >> 6
	/* Partial preparation is 100 times slower (per socket). */
	if oneShot && inputSize <= partialPrepareThreshold {
		var i uint
		for i = 0; i < inputSize; i++ {
			var bucket uint = h.HashBytes(data[i:])

			/* See InitEmpty comment. */
			h.addr[bucket] = 0xCCCCCCCC

			h.head[bucket] = 0xCCCC
		}
	} else {
		/* Fill |addr| array with 0xCCCCCCCC value. Because of wrapping, position
		   processed by hasher never reaches 3GB + 64M; this makes all new chains
		   to be terminated after the first node. */
		for i := range h.addr {
			h.addr[i] = 0xCCCCCCCC
		}

		for i := range h.head {
			h.head[i] = 0
		}
	}

	h.tinyHash = [65536]byte{}
	for i := range h.freeSlotIdx {
		h.freeSlotIdx[i] = 0
	}
}

/*
Look at 4 bytes at &data[ix & mask]. Compute a hash from these, and prepend

	node to corresponding chain; also update tiny_hash for current position.
*/
func (h *hashForgetfulChain) Store(data []byte, mask uint, ix uint) {
	var key uint = h.HashBytes(data[ix&mask:])
	var bank uint = key & (h.numBanks - 1)
	idx := uint(h.freeSlotIdx[bank]) & ((1 << h.bankBits) - 1)
	h.freeSlotIdx[bank]++
	var delta uint = ix - uint(h.addr[key])
	h.tinyHash[uint16(ix)] = byte(key)
	if delta > 0xFFFF {
		delta = 0xFFFF
	}
	h.banks[bank][idx].delta = uint16(delta)
	h.banks[bank][idx].next = h.head[key]
	h.addr[key] = uint32(ix)
	h.head[key] = uint16(idx)
}

func (h *hashForgetfulChain) StoreRange(data []byte, mask uint, ixStart uint, ixEnd uint) {
	var i uint
	for i = ixStart; i < ixEnd; i++ {
		h.Store(data, mask, i)
	}
}

func (h *hashForgetfulChain) StitchToPreviousBlock(numBytes uint, position uint, ringbuffer []byte, ringBufferMask uint) {
	if numBytes >= h.HashTypeLength()-1 && position >= 3 {
		/* Prepare the hashes for three last bytes of the last write.
		   These could not be calculated before, since they require knowledge
		   of both the previous and the current block. */
		h.Store(ringbuffer, ringBufferMask, position-3)
		h.Store(ringbuffer, ringBufferMask, position-2)
		h.Store(ringbuffer, ringBufferMask, position-1)
	}
}

func (h *hashForgetfulChain) PrepareDistanceCache(distanceCache []int) {
	prepareDistanceCache(distanceCache, h.numLastDistancesToCheck)
}

/*
Find a longest backward match of &data[cur_ix] up to the length of

	max_length and stores the position cur_ix in the hash table.

	REQUIRES: PrepareDistanceCachehashForgetfulChain must be invoked for current distance cache
	          values; if this method is invoked repeatedly with the same distance
	          cache values, it is enough to invoke PrepareDistanceCachehashForgetfulChain once.

	Does not look for matches longer than max_length.
	Does not look for matches further away than max_backward.
	Writes the best match into |out|.
	|out|->score is updated only if a better match is found.
*/
func (h *hashForgetfulChain) FindLongestMatch(dictionary *encoderDictionary, data []byte, ringBufferMask uint, distanceCache []int, curIx uint, maxLength uint, maxBackward uint, gap uint, maxDistance uint, out *hasherSearchResult) {
	var curIxMasked uint = curIx & ringBufferMask
	var minScore uint = out.score
	var bestScore uint = out.score
	var bestLen uint = out.len
	var key uint = h.HashBytes(data[curIxMasked:])
	var tinyHash byte = byte(key)
	/* Don't accept a short copy from far away. */
	out.len = 0

	out.len_code_delta = 0

	/* Try last distance first. */
	for i := 0; i < h.numLastDistancesToCheck; i++ {
		var backward uint = uint(distanceCache[i])
		var prevIx uint = curIx - backward

		/* For distance code 0 we want to consider 2-byte matches. */
		if i > 0 && h.tinyHash[uint16(prevIx)] != tinyHash {
			continue
		}
		if prevIx >= curIx || backward > maxBackward {
			continue
		}

		prevIx &= ringBufferMask
		{
			var limit uint = findMatchLengthWithLimit(data[prevIx:], data[curIxMasked:], maxLength)
			if limit >= 2 {
				var score uint = backwardReferenceScoreUsingLastDistance(uint(limit))
				if bestScore < score {
					if i != 0 {
						score -= backwardReferencePenaltyUsingLastDistance(uint(i))
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
		var bank uint = key & (h.numBanks - 1)
		var backward uint = 0
		var hops uint = h.maxHops
		var delta uint = curIx - uint(h.addr[key])
		var slot uint = uint(h.head[key])
		for {
			tmp6 := hops
			hops--
			if tmp6 == 0 {
				break
			}
			var prevIx uint
			var last uint = slot
			backward += delta
			if backward > maxBackward {
				break
			}
			prevIx = (curIx - backward) & ringBufferMask
			slot = uint(h.banks[bank][last].next)
			delta = uint(h.banks[bank][last].delta)
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

		h.Store(data, ringBufferMask, curIx)
	}

	if out.score == minScore {
		searchInStaticDictionary(dictionary, h, data[curIxMasked:], maxLength, maxBackward+gap, maxDistance, out, false)
	}
}
