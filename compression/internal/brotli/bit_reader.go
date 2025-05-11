package brotli

import "encoding/binary"

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* A bit reading helpers */

const shortFillBitWindowRead = 8 >> 1

var kBitMask = [33]uint32{
	0x00000000,
	0x00000001,
	0x00000003,
	0x00000007,
	0x0000000F,
	0x0000001F,
	0x0000003F,
	0x0000007F,
	0x000000FF,
	0x000001FF,
	0x000003FF,
	0x000007FF,
	0x00000FFF,
	0x00001FFF,
	0x00003FFF,
	0x00007FFF,
	0x0000FFFF,
	0x0001FFFF,
	0x0003FFFF,
	0x0007FFFF,
	0x000FFFFF,
	0x001FFFFF,
	0x003FFFFF,
	0x007FFFFF,
	0x00FFFFFF,
	0x01FFFFFF,
	0x03FFFFFF,
	0x07FFFFFF,
	0x0FFFFFFF,
	0x1FFFFFFF,
	0x3FFFFFFF,
	0x7FFFFFFF,
	0xFFFFFFFF,
}

func bitMask(n uint32) uint32 {
	return kBitMask[n]
}

type bitReader struct {
	val      uint64
	bitPos   uint32
	input    []byte
	inputLen uint
	bytePos  uint
}

type bitReaderState struct {
	val      uint64
	bitPos   uint32
	input    []byte
	inputLen uint
	bytePos  uint
}

/* Initializes the BrotliBitReader fields. */

/*
Ensures that accumulator is not empty.

	May consume up to sizeof(brotli_reg_t) - 1 bytes of input.
	Returns false if data is required but there is no input available.
	For BROTLI_ALIGNED_READ this function also prepares bit reader for aligned
	reading.
*/
func bitReaderSaveState(from *bitReader, to *bitReaderState) {
	to.val = from.val
	to.bitPos = from.bitPos
	to.input = from.input
	to.inputLen = from.inputLen
	to.bytePos = from.bytePos
}

func bitReaderRestoreState(to *bitReader, from *bitReaderState) {
	to.val = from.val
	to.bitPos = from.bitPos
	to.input = from.input
	to.inputLen = from.inputLen
	to.bytePos = from.bytePos
}

func getAvailableBits(br *bitReader) uint32 {
	return 64 - br.bitPos
}

/*
Returns amount of unread bytes the bit reader still has buffered from the

	BrotliInput, including whole bytes in br->val.
*/
func getRemainingBytes(br *bitReader) uint {
	return uint(uint32(br.inputLen-br.bytePos) + (getAvailableBits(br) >> 3))
}

/*
Checks if there is at least |num| bytes left in the input ring-buffer

	(excluding the bits remaining in br->val).
*/
func checkInputAmount(br *bitReader, num uint) bool {
	return br.inputLen-br.bytePos >= num
}

/*
Guarantees that there are at least |n_bits| + 1 bits in accumulator.

	Precondition: accumulator contains at least 1 bit.
	|n_bits| should be in the range [1..24] for regular build. For portable
	non-64-bit little-endian build, only 16 bits are safe to request.
*/
func fillBitWindow(br *bitReader, _ uint32) {
	if br.bitPos >= 32 {
		br.val >>= 32
		br.bitPos ^= 32 /* here same as -= 32 because of the if condition */
		br.val |= (uint64(binary.LittleEndian.Uint32(br.input[br.bytePos:]))) << 32
		br.bytePos += 4
	}
}

/*
Mostly like BrotliFillBitWindow, but guarantees only 16 bits and reads no

	more than BROTLI_SHORT_FILL_BIT_WINDOW_READ bytes of input.
*/
func fillBitWindow16(br *bitReader) {
	fillBitWindow(br, 17)
}

/*
Tries to pull one byte of input to accumulator.

	Returns false if there is no input available.
*/
func pullByte(br *bitReader) bool {
	if br.bytePos == br.inputLen {
		return false
	}

	br.val >>= 8
	br.val |= (uint64(br.input[br.bytePos])) << 56
	br.bitPos -= 8
	br.bytePos++
	return true
}

/*
Returns currently available bits.

	The number of valid bits could be calculated by BrotliGetAvailableBits.
*/
func getBitsUnmasked(br *bitReader) uint64 {
	return br.val >> br.bitPos
}

/*
Like BrotliGetBits, but does not mask the result.

	The result contains at least 16 valid bits.
*/
func get16BitsUnmasked(br *bitReader) uint32 {
	fillBitWindow(br, 16)
	return uint32(getBitsUnmasked(br))
}

/*
Returns the specified number of bits from |br| without advancing bit

	position.
*/
func getBits(br *bitReader, nBits uint32) uint32 {
	fillBitWindow(br, nBits)
	return uint32(getBitsUnmasked(br)) & bitMask(nBits)
}

/*
Tries to peek the specified amount of bits. Returns false, if there

	is not enough input.
*/
func safeGetBits(br *bitReader, nBits uint32, val *uint32) bool {
	for getAvailableBits(br) < nBits {
		if !pullByte(br) {
			return false
		}
	}

	*val = uint32(getBitsUnmasked(br)) & bitMask(nBits)
	return true
}

/* Advances the bit pos by |n_bits|. */
func dropBits(br *bitReader, nBits uint32) {
	br.bitPos += nBits
}

func bitReaderUnload(br *bitReader) {
	var unusedBytes = getAvailableBits(br) >> 3
	var unusedBits = unusedBytes << 3
	br.bytePos -= uint(unusedBytes)
	if unusedBits == 64 {
		br.val = 0
	} else {
		br.val <<= unusedBits
	}

	br.bitPos += unusedBits
}

/*
Reads the specified number of bits from |br| and advances the bit pos.

	Precondition: accumulator MUST contain at least |n_bits|.
*/
func takeBits(br *bitReader, nBits uint32, val *uint32) {
	*val = uint32(getBitsUnmasked(br)) & bitMask(nBits)
	dropBits(br, nBits)
}

/*
Reads the specified number of bits from |br| and advances the bit pos.

	Assumes that there is enough input to perform BrotliFillBitWindow.
*/
func readBits(br *bitReader, nBits uint32) uint32 {
	var val uint32
	fillBitWindow(br, nBits)
	takeBits(br, nBits, &val)
	return val
}

/*
Tries to read the specified amount of bits. Returns false, if there

	is not enough input. |n_bits| MUST be positive.
*/
func safeReadBits(br *bitReader, nBits uint32, val *uint32) bool {
	for getAvailableBits(br) < nBits {
		if !pullByte(br) {
			return false
		}
	}

	takeBits(br, nBits, val)
	return true
}

/*
Advances the bit reader position to the next byte boundary and verifies

	that any skipped bits are set to zero.
*/
func bitReaderJumpToByteBoundary(br *bitReader) bool {
	var padBitsCount = getAvailableBits(br) & 0x7
	var padBits uint32 = 0
	if padBitsCount != 0 {
		takeBits(br, padBitsCount, &padBits)
	}

	return padBits == 0
}

/*
Copies the remaining input bytes stored in the bit reader to the output. Value

	|num| may not be larger than BrotliGetRemainingBytes. The bit reader must be
	warmed up again after this.
*/
func copyBytes(dest []byte, br *bitReader, num uint) {
	for getAvailableBits(br) >= 8 && num > 0 {
		dest[0] = byte(getBitsUnmasked(br))
		dropBits(br, 8)
		dest = dest[1:]
		num--
	}

	copy(dest, br.input[br.bytePos:][:num])
	br.bytePos += num
}

func initBitReader(br *bitReader) {
	br.val = 0
	br.bitPos = 64
}

func warmupBitReader(br *bitReader) bool {
	/* Fixing alignment after unaligned BrotliFillWindow would result accumulator
	   overflow. If unalignment is caused by BrotliSafeReadBits, then there is
	   enough space in accumulator to fix alignment. */
	if getAvailableBits(br) == 0 {
		if !pullByte(br) {
			return false
		}
	}

	return true
}
