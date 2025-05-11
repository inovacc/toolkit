package brotli

/* Copyright 2018 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* NOTE: this hasher does not search in the dictionary. It is used as
   backup-hasher, the main hasher already searches in it. */

const kRollingHashMul32 uint32 = 69069

const kInvalidPosHashRolling uint32 = 0xffffffff

/*
This hasher uses a longer forward length, but returning a higher value here

	will hurt compression by the main hasher when combined with a composite
	hasher. The hasher tests for forward itself instead.
*/
func (*hashRolling) HashTypeLength() uint {
	return 4
}

func (*hashRolling) StoreLookahead() uint {
	return 4
}

/*
Computes a code from a single byte. A lookup table of 256 values could be

	used, but simply adding 1 works about as good.
*/
func (*hashRolling) HashByte(b byte) uint32 {
	return uint32(b) + 1
}

func (h *hashRolling) HashRollingFunctionInitial(state uint32, add byte, factor uint32) uint32 {
	return uint32(factor*state + h.HashByte(add))
}

func (h *hashRolling) HashRollingFunction(state uint32, add byte, rem byte, factor uint32, factor_remove uint32) uint32 {
	return uint32(factor*state + h.HashByte(add) - factor_remove*h.HashByte(rem))
}

/*
Rolling hash for long distance long string matches. Stores one position

	per bucket, bucket key is computed over a long region.
*/
type hashRolling struct {
	hasherCommon

	jump int

	state        uint32
	table        []uint32
	nextIx       uint
	factor       uint32
	factorRemove uint32
}

func (h *hashRolling) Initialize(params *encoderParams) {
	h.state = 0
	h.nextIx = 0

	h.factor = kRollingHashMul32

	/* Compute the factor of the oldest byte to remove: factor**steps modulo
	   0xffffffff (the multiplications rely on 32-bit overflow) */
	h.factorRemove = 1

	for i := 0; i < 32; i += h.jump {
		h.factorRemove *= h.factor
	}

	h.table = make([]uint32, 16777216)
	for i := 0; i < 16777216; i++ {
		h.table[i] = kInvalidPosHashRolling
	}
}

func (h *hashRolling) Prepare(_ bool, inputSize uint, data []byte) {
	/* Too small size, cannot use this hasher. */
	if inputSize < 32 {
		return
	}
	h.state = 0
	for i := 0; i < 32; i += h.jump {
		h.state = h.HashRollingFunctionInitial(h.state, data[i], h.factor)
	}
}

func (*hashRolling) Store(data []byte, mask uint, ix uint) {
}

func (*hashRolling) StoreRange(data []byte, mask uint, ix_start uint, ix_end uint) {
}

func (h *hashRolling) StitchToPreviousBlock(numBytes uint, position uint, ringbuffer []byte, ringBufferMask uint) {
	var positionMasked uint
	/* In this case we must re-initialize the hasher from scratch from the
	   current position. */

	var available uint = numBytes
	if position&uint(h.jump-1) != 0 {
		var diff uint = uint(h.jump) - (position & uint(h.jump-1))
		if diff > available {
			available = 0
		} else {
			available = available - diff
		}
		position += diff
	}

	positionMasked = position & ringBufferMask

	/* wrapping around ringbuffer not handled. */
	if available > ringBufferMask-positionMasked {
		available = ringBufferMask - positionMasked
	}

	h.Prepare(false, available, ringbuffer[position&ringBufferMask:])
	h.nextIx = position
}

func (*hashRolling) PrepareDistanceCache(distance_cache []int) {
}

func (h *hashRolling) FindLongestMatch(dictionary *encoderDictionary, data []byte, ringBufferMask uint, _ []int, curIx uint, maxLength uint, maxBackward uint, _ uint, _ uint, out *hasherSearchResult) {
	var curIxMasked uint = curIx & ringBufferMask
	var pos uint = h.nextIx

	if curIx&uint(h.jump-1) != 0 {
		return
	}

	/* Not enough lookahead */
	if maxLength < 32 {
		return
	}

	for pos = h.nextIx; pos <= curIx; pos += uint(h.jump) {
		var code uint32 = h.state & ((16777216 * 64) - 1)
		var rem byte = data[pos&ringBufferMask]
		var add byte = data[(pos+32)&ringBufferMask]
		var foundIx uint = uint(kInvalidPosHashRolling)

		h.state = h.HashRollingFunction(h.state, add, rem, h.factor, h.factorRemove)

		if code < 16777216 {
			foundIx = uint(h.table[code])
			h.table[code] = uint32(pos)
			if pos == curIx && uint32(foundIx) != kInvalidPosHashRolling {
				/* The cast to 32-bit makes backward distances up to 4GB work even
				   if cur_ix is above 4GB, despite using 32-bit values in the table. */
				var backward uint = uint(uint32(curIx - foundIx))
				if backward <= maxBackward {
					var foundIxMasked uint = foundIx & ringBufferMask
					var limit uint = findMatchLengthWithLimit(data[foundIxMasked:], data[curIxMasked:], maxLength)
					if limit >= 4 && limit > out.len {
						var score uint = backwardReferenceScore(uint(limit), backward)
						if score > out.score {
							out.len = uint(limit)
							out.distance = backward
							out.score = score
							out.len_code_delta = 0
						}
					}
				}
			}
		}
	}

	h.nextIx = curIx + uint(h.jump)
}
