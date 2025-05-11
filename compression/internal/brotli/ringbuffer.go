package brotli

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/*
A ringBuffer(window_bits, tail_bits) contains `1 << window_bits' bytes of

	data in a circular manner: writing a byte writes it to:
	  `position() % (1 << window_bits)'.
	For convenience, the ringBuffer array contains another copy of the
	first `1 << tail_bits' bytes:
	  buffer_[i] == buffer_[i + (1 << window_bits)], if i < (1 << tail_bits),
	and another copy of the last two bytes:
	  buffer_[-1] == buffer_[(1 << window_bits) - 1] and
	  buffer_[-2] == buffer_[(1 << window_bits) - 2].
*/
type ringBuffer struct {
	size_     uint32
	mask_     uint32
	tailSize  uint32
	totalSize uint32
	curSize   uint32
	pos_      uint32
	data_     []byte
	buffer_   []byte
}

func ringBufferInit(rb *ringBuffer) {
	rb.pos_ = 0
}

func ringBufferSetup(params *encoderParams, rb *ringBuffer) {
	var windowBits int = computeRbBits(params)
	var tailBits int = params.lgblock
	*(*uint32)(&rb.size_) = 1 << uint(windowBits)
	*(*uint32)(&rb.mask_) = (1 << uint(windowBits)) - 1
	*(*uint32)(&rb.tailSize) = 1 << uint(tailBits)
	*(*uint32)(&rb.totalSize) = rb.size_ + rb.tailSize
}

const kSlackForEightByteHashingEverywhere uint = 7

/*
Allocates or re-allocates data_ to the given length + plus some slack

	region before and after. Fills the slack regions with zeros.
*/
func ringBufferInitBuffer(buflen uint32, rb *ringBuffer) {
	var newData []byte
	var i uint
	size := 2 + int(buflen) + int(kSlackForEightByteHashingEverywhere)
	if cap(rb.data_) < size {
		newData = make([]byte, size)
	} else {
		newData = rb.data_[:size]
	}
	if rb.data_ != nil {
		copy(newData, rb.data_[:2+rb.curSize+uint32(kSlackForEightByteHashingEverywhere)])
	}

	rb.data_ = newData
	rb.curSize = buflen
	rb.buffer_ = rb.data_[2:]
	rb.data_[1] = 0
	rb.data_[0] = rb.data_[1]
	for i = 0; i < kSlackForEightByteHashingEverywhere; i++ {
		rb.buffer_[rb.curSize+uint32(i)] = 0
	}
}

func ringBufferWriteTail(bytes []byte, n uint, rb *ringBuffer) {
	var maskedPos uint = uint(rb.pos_ & rb.mask_)
	if uint32(maskedPos) < rb.tailSize {
		/* Just fill the tail buffer with the beginning data. */
		var p uint = uint(rb.size_ + uint32(maskedPos))
		copy(rb.buffer_[p:], bytes[:brotliMinSizeT(n, uint(rb.tailSize-uint32(maskedPos)))])
	}
}

/* Push bytes into the ring buffer. */
func ringBufferWrite(bytes []byte, n uint, rb *ringBuffer) {
	if rb.pos_ == 0 && uint32(n) < rb.tailSize {
		/* Special case for the first write: to process the first block, we don't
		   need to allocate the whole ring-buffer and we don't need the tail
		   either. However, we do this memory usage optimization only if the
		   first write is less than the tail size, which is also the input block
		   size, otherwise it is likely that other blocks will follow and we
		   will need to reallocate to the full size anyway. */
		rb.pos_ = uint32(n)

		ringBufferInitBuffer(rb.pos_, rb)
		copy(rb.buffer_, bytes[:n])
		return
	}

	if rb.curSize < rb.totalSize {
		/* Lazily allocate the full buffer. */
		ringBufferInitBuffer(rb.totalSize, rb)

		/* Initialize the last two bytes to zero, so that we don't have to worry
		   later when we copy the last two bytes to the first two positions. */
		rb.buffer_[rb.size_-2] = 0

		rb.buffer_[rb.size_-1] = 0
	}
	{
		var maskedPos uint = uint(rb.pos_ & rb.mask_)

		/* The length of the writes is limited so that we do not need to worry
		   about a write */
		ringBufferWriteTail(bytes, n, rb)

		if uint32(maskedPos+n) <= rb.size_ {
			/* A single write fits. */
			copy(rb.buffer_[maskedPos:], bytes[:n])
		} else {
			/* Split into two writes.
			   Copy into the end of the buffer, including the tail buffer. */
			copy(rb.buffer_[maskedPos:], bytes[:brotliMinSizeT(n, uint(rb.totalSize-uint32(maskedPos)))])

			/* Copy into the beginning of the buffer */
			copy(rb.buffer_, bytes[rb.size_-uint32(maskedPos):][:uint32(n)-(rb.size_-uint32(maskedPos))])
		}
	}
	{
		var notFirstLap bool = rb.pos_&(1<<31) != 0
		var rbPosMask uint32 = (1 << 31) - 1
		rb.data_[0] = rb.buffer_[rb.size_-2]
		rb.data_[1] = rb.buffer_[rb.size_-1]
		rb.pos_ = (rb.pos_ & rbPosMask) + uint32(uint32(n)&rbPosMask)
		if notFirstLap {
			/* Wrap, but preserve not-a-first-lap feature. */
			rb.pos_ |= 1 << 31
		}
	}
}
