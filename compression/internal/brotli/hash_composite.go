package brotli

/* Copyright 2018 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

func (h *hashComposite) HashTypeLength() uint {
	var a uint = h.ha.HashTypeLength()
	var b uint = h.hb.HashTypeLength()
	if a > b {
		return a
	} else {
		return b
	}
}

func (h *hashComposite) StoreLookahead() uint {
	var a uint = h.ha.StoreLookahead()
	var b uint = h.hb.StoreLookahead()
	if a > b {
		return a
	} else {
		return b
	}
}

/*
Composite hasher: This hasher allows to combine two other hashers, HASHER_A

	and HASHER_B.
*/
type hashComposite struct {
	hasherCommon
	ha     hasherHandle
	hb     hasherHandle
	params *encoderParams
}

func (h *hashComposite) Initialize(params *encoderParams) {
	h.params = params
}

/*
TODO: Initialize of the hashers is defered to Prepare (and params

	remembered here) because we don't get the one_shot and input_size params
	here that are needed to know the memory size of them. Instead provide
	those params to all hashers InitializehashComposite
*/
func (h *hashComposite) Prepare(oneShot bool, inputSize uint, data []byte) {
	if h.ha == nil {
		var commonA *hasherCommon
		var commonB *hasherCommon

		commonA = h.ha.Common()
		commonA.params = h.params.hasher
		commonA.is_prepared_ = false
		commonA.dict_num_lookups = 0
		commonA.dict_num_matches = 0
		h.ha.Initialize(h.params)

		commonB = h.hb.Common()
		commonB.params = h.params.hasher
		commonB.is_prepared_ = false
		commonB.dict_num_lookups = 0
		commonB.dict_num_matches = 0
		h.hb.Initialize(h.params)
	}

	h.ha.Prepare(oneShot, inputSize, data)
	h.hb.Prepare(oneShot, inputSize, data)
}

func (h *hashComposite) Store(data []byte, mask uint, ix uint) {
	h.ha.Store(data, mask, ix)
	h.hb.Store(data, mask, ix)
}

func (h *hashComposite) StoreRange(data []byte, mask uint, ixStart uint, ixEnd uint) {
	h.ha.StoreRange(data, mask, ixStart, ixEnd)
	h.hb.StoreRange(data, mask, ixStart, ixEnd)
}

func (h *hashComposite) StitchToPreviousBlock(numBytes uint, position uint, ringbuffer []byte, ringBufferMask uint) {
	h.ha.StitchToPreviousBlock(numBytes, position, ringbuffer, ringBufferMask)
	h.hb.StitchToPreviousBlock(numBytes, position, ringbuffer, ringBufferMask)
}

func (h *hashComposite) PrepareDistanceCache(distanceCache []int) {
	h.ha.PrepareDistanceCache(distanceCache)
	h.hb.PrepareDistanceCache(distanceCache)
}

func (h *hashComposite) FindLongestMatch(dictionary *encoderDictionary, data []byte, ringBufferMask uint, distanceCache []int, curIx uint, maxLength uint, maxBackward uint, gap uint, maxDistance uint, out *hasherSearchResult) {
	h.ha.FindLongestMatch(dictionary, data, ringBufferMask, distanceCache, curIx, maxLength, maxBackward, gap, maxDistance, out)
	h.hb.FindLongestMatch(dictionary, data, ringBufferMask, distanceCache, curIx, maxLength, maxBackward, gap, maxDistance, out)
}
