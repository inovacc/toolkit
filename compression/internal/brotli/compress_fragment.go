package brotli

import "encoding/binary"

/* Copyright 2015 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Function for fast encoding of an input fragment, independently from the input
   history. This function uses one-pass processing: when we find a backward
   match, we immediately emit the corresponding command and literal codes to
   the bit stream.

   Adapted from the CompressFragment() function in
   https://github.com/google/snappy/blob/master/snappy.cc */

const maxdistanceCompressFragment = 262128

func hash5(p []byte, shift uint) uint32 {
	var h uint64 = (binary.LittleEndian.Uint64(p) << 24) * uint64(kHashMul32)
	return uint32(h >> shift)
}

func hashBytesAtOffset5(v uint64, offset int, shift uint) uint32 {
	assert(offset >= 0)
	assert(offset <= 3)
	{
		var h uint64 = ((v >> uint(8*offset)) << 24) * uint64(kHashMul32)
		return uint32(h >> shift)
	}
}

func isMatch5(p1 []byte, p2 []byte) bool {
	return binary.LittleEndian.Uint32(p1) == binary.LittleEndian.Uint32(p2) &&
		p1[4] == p2[4]
}

/*
Builds a literal prefix code into "depths" and "bits" based on the statistics

	of the "input" string and stores it into the bit stream.
	Note that the prefix code here is built from the pre-LZ77 input, therefore
	we can only approximate the statistics of the actual literal stream.
	Moreover, for long inputs we build a histogram from a sample of the input
	and thus have to assign a non-zero depth for each literal.
	Returns estimated compression ratio millibytes/char for encoding given input
	with generated code.
*/
func buildAndStoreLiteralPrefixCode(input []byte, inputSize uint, depths []byte, bits []uint16, storageIx *uint, storage []byte) uint {
	var histogram = [256]uint32{0}
	var histogramTotal uint
	var i uint
	if inputSize < 1<<15 {
		for i = 0; i < inputSize; i++ {
			histogram[input[i]]++
		}

		histogramTotal = inputSize
		for i = 0; i < 256; i++ {
			/* We weigh the first 11 samples with weight 3 to account for the
			   balancing effect of the LZ77 phase on the histogram. */
			var adjust uint32 = 2 * brotliMinUint32T(histogram[i], 11)
			histogram[i] += adjust
			histogramTotal += uint(adjust)
		}
	} else {
		const kSampleRate uint = 29
		for i = 0; i < inputSize; i += kSampleRate {
			histogram[input[i]]++
		}

		histogramTotal = (inputSize + kSampleRate - 1) / kSampleRate
		for i = 0; i < 256; i++ {
			/* We add 1 to each population count to avoid 0 bit depths (since this is
			   only a sample and we don't know if the symbol appears or not), and we
			   weigh the first 11 samples with weight 3 to account for the balancing
			   effect of the LZ77 phase on the histogram (more frequent symbols are
			   more likely to be in backward references instead as literals). */
			var adjust uint32 = 1 + 2*brotliMinUint32T(histogram[i], 11)
			histogram[i] += adjust
			histogramTotal += uint(adjust)
		}
	}

	buildAndStoreHuffmanTreeFast(histogram[:], histogramTotal, /* max_bits = */
		8, depths, bits, storageIx, storage)
	{
		var literalRatio uint = 0
		for i = 0; i < 256; i++ {
			if histogram[i] != 0 {
				literalRatio += uint(histogram[i] * uint32(depths[i]))
			}
		}

		/* Estimated encoding ratio, millibytes per symbol. */
		return (literalRatio * 125) / histogramTotal
	}
}

/*
Builds a command and distance prefix code (each 64 symbols) into "depth" and

	"bits" based on "histogram" and stores it into the bit stream.
*/
func buildAndStoreCommandPrefixCode1(histogram []uint32, depth []byte, bits []uint16, storageIx *uint, storage []byte) {
	var tree [129]huffmanTree
	var cmdDepth = [numCommandSymbols]byte{0}
	/* Tree size for building a tree over 64 symbols is 2 * 64 + 1. */

	var cmdBits [64]uint16

	createHuffmanTree(histogram, 64, 15, tree[:], depth)
	createHuffmanTree(histogram[64:], 64, 14, tree[:], depth[64:])

	/* We have to jump through a few hoops here in order to compute
	   the command bits because the symbols are in a different order than in
	   the full alphabet. This looks complicated, but having the symbols
	   in this order in the command bits saves a few branches in the Emit*
	   functions. */
	copy(cmdDepth[:], depth[:24])

	copy(cmdDepth[24:][:], depth[40:][:8])
	copy(cmdDepth[32:][:], depth[24:][:8])
	copy(cmdDepth[40:][:], depth[48:][:8])
	copy(cmdDepth[48:][:], depth[32:][:8])
	copy(cmdDepth[56:][:], depth[56:][:8])
	convertBitDepthsToSymbols(cmdDepth[:], 64, cmdBits[:])
	copy(bits, cmdBits[:24])
	copy(bits[24:], cmdBits[32:][:8])
	copy(bits[32:], cmdBits[48:][:8])
	copy(bits[40:], cmdBits[24:][:8])
	copy(bits[48:], cmdBits[40:][:8])
	copy(bits[56:], cmdBits[56:][:8])
	convertBitDepthsToSymbols(depth[64:], 64, bits[64:])
	{
		/* Create the bit length array for the full command alphabet. */
		var i uint
		for i := 0; i < int(64); i++ {
			cmdDepth[i] = 0
		} /* only 64 first values were used */
		copy(cmdDepth[:], depth[:8])
		copy(cmdDepth[64:][:], depth[8:][:8])
		copy(cmdDepth[128:][:], depth[16:][:8])
		copy(cmdDepth[192:][:], depth[24:][:8])
		copy(cmdDepth[384:][:], depth[32:][:8])
		for i = 0; i < 8; i++ {
			cmdDepth[128+8*i] = depth[40+i]
			cmdDepth[256+8*i] = depth[48+i]
			cmdDepth[448+8*i] = depth[56+i]
		}

		storeHuffmanTree(cmdDepth[:], numCommandSymbols, tree[:], storageIx, storage)
	}

	storeHuffmanTree(depth[64:], 64, tree[:], storageIx, storage)
}

/* REQUIRES: insertlen < 6210 */
func emitInsertLen1(insertlen uint, depth []byte, bits []uint16, histo []uint32, storageIx *uint, storage []byte) {
	if insertlen < 6 {
		var code uint = insertlen + 40
		writeBits(uint(depth[code]), uint64(bits[code]), storageIx, storage)
		histo[code]++
	} else if insertlen < 130 {
		var tail uint = insertlen - 2
		var nbits uint32 = log2FloorNonZero(tail) - 1
		var prefix uint = tail >> nbits
		var inscode uint = uint((nbits << 1) + uint32(prefix) + 42)
		writeBits(uint(depth[inscode]), uint64(bits[inscode]), storageIx, storage)
		writeBits(uint(nbits), uint64(tail)-(uint64(prefix)<<nbits), storageIx, storage)
		histo[inscode]++
	} else if insertlen < 2114 {
		var tail uint = insertlen - 66
		var nbits uint32 = log2FloorNonZero(tail)
		var code uint = uint(nbits + 50)
		writeBits(uint(depth[code]), uint64(bits[code]), storageIx, storage)
		writeBits(uint(nbits), uint64(tail)-(uint64(uint(1))<<nbits), storageIx, storage)
		histo[code]++
	} else {
		writeBits(uint(depth[61]), uint64(bits[61]), storageIx, storage)
		writeBits(12, uint64(insertlen)-2114, storageIx, storage)
		histo[61]++
	}
}

func emitLongInsertLen(insertlen uint, depth []byte, bits []uint16, histo []uint32, storage_ix *uint, storage []byte) {
	if insertlen < 22594 {
		writeBits(uint(depth[62]), uint64(bits[62]), storage_ix, storage)
		writeBits(14, uint64(insertlen)-6210, storage_ix, storage)
		histo[62]++
	} else {
		writeBits(uint(depth[63]), uint64(bits[63]), storage_ix, storage)
		writeBits(24, uint64(insertlen)-22594, storage_ix, storage)
		histo[63]++
	}
}

func emitCopyLen1(copylen uint, depth []byte, bits []uint16, histo []uint32, storage_ix *uint, storage []byte) {
	if copylen < 10 {
		writeBits(uint(depth[copylen+14]), uint64(bits[copylen+14]), storage_ix, storage)
		histo[copylen+14]++
	} else if copylen < 134 {
		var tail uint = copylen - 6
		var nbits uint32 = log2FloorNonZero(tail) - 1
		var prefix uint = tail >> nbits
		var code uint = uint((nbits << 1) + uint32(prefix) + 20)
		writeBits(uint(depth[code]), uint64(bits[code]), storage_ix, storage)
		writeBits(uint(nbits), uint64(tail)-(uint64(prefix)<<nbits), storage_ix, storage)
		histo[code]++
	} else if copylen < 2118 {
		var tail uint = copylen - 70
		var nbits uint32 = log2FloorNonZero(tail)
		var code uint = uint(nbits + 28)
		writeBits(uint(depth[code]), uint64(bits[code]), storage_ix, storage)
		writeBits(uint(nbits), uint64(tail)-(uint64(uint(1))<<nbits), storage_ix, storage)
		histo[code]++
	} else {
		writeBits(uint(depth[39]), uint64(bits[39]), storage_ix, storage)
		writeBits(24, uint64(copylen)-2118, storage_ix, storage)
		histo[39]++
	}
}

func emitCopyLenLastDistance1(copylen uint, depth []byte, bits []uint16, histo []uint32, storageIx *uint, storage []byte) {
	if copylen < 12 {
		writeBits(uint(depth[copylen-4]), uint64(bits[copylen-4]), storageIx, storage)
		histo[copylen-4]++
	} else if copylen < 72 {
		var tail uint = copylen - 8
		var nbits uint32 = log2FloorNonZero(tail) - 1
		var prefix uint = tail >> nbits
		var code uint = uint((nbits << 1) + uint32(prefix) + 4)
		writeBits(uint(depth[code]), uint64(bits[code]), storageIx, storage)
		writeBits(uint(nbits), uint64(tail)-(uint64(prefix)<<nbits), storageIx, storage)
		histo[code]++
	} else if copylen < 136 {
		var tail uint = copylen - 8
		var code uint = (tail >> 5) + 30
		writeBits(uint(depth[code]), uint64(bits[code]), storageIx, storage)
		writeBits(5, uint64(tail)&31, storageIx, storage)
		writeBits(uint(depth[64]), uint64(bits[64]), storageIx, storage)
		histo[code]++
		histo[64]++
	} else if copylen < 2120 {
		var tail uint = copylen - 72
		var nbits uint32 = log2FloorNonZero(tail)
		var code uint = uint(nbits + 28)
		writeBits(uint(depth[code]), uint64(bits[code]), storageIx, storage)
		writeBits(uint(nbits), uint64(tail)-(uint64(uint(1))<<nbits), storageIx, storage)
		writeBits(uint(depth[64]), uint64(bits[64]), storageIx, storage)
		histo[code]++
		histo[64]++
	} else {
		writeBits(uint(depth[39]), uint64(bits[39]), storageIx, storage)
		writeBits(24, uint64(copylen)-2120, storageIx, storage)
		writeBits(uint(depth[64]), uint64(bits[64]), storageIx, storage)
		histo[39]++
		histo[64]++
	}
}

func emitDistance1(distance uint, depth []byte, bits []uint16, histo []uint32, storageIx *uint, storage []byte) {
	var d uint = distance + 3
	var nbits uint32 = log2FloorNonZero(d) - 1
	var prefix uint = (d >> nbits) & 1
	var offset uint = (2 + prefix) << nbits
	var distcode uint = uint(2*(nbits-1) + uint32(prefix) + 80)
	writeBits(uint(depth[distcode]), uint64(bits[distcode]), storageIx, storage)
	writeBits(uint(nbits), uint64(d)-uint64(offset), storageIx, storage)
	histo[distcode]++
}

func emitLiterals(input []byte, len uint, depth []byte, bits []uint16, storageIx *uint, storage []byte) {
	var j uint
	for j = 0; j < len; j++ {
		var lit byte = input[j]
		writeBits(uint(depth[lit]), uint64(bits[lit]), storageIx, storage)
	}
}

/* REQUIRES: len <= 1 << 24. */
func storeMetaBlockHeader1(len uint, isUncompressed bool, storageIx *uint, storage []byte) {
	var nibbles uint = 6

	/* ISLAST */
	writeBits(1, 0, storageIx, storage)

	if len <= 1<<16 {
		nibbles = 4
	} else if len <= 1<<20 {
		nibbles = 5
	}

	writeBits(2, uint64(nibbles)-4, storageIx, storage)
	writeBits(nibbles*4, uint64(len)-1, storageIx, storage)

	/* ISUNCOMPRESSED */
	writeSingleBit(isUncompressed, storageIx, storage)
}

func updateBits(n_bits uint, bits uint32, pos uint, array []byte) {
	for n_bits > 0 {
		var bytePos uint = pos >> 3
		var nUnchangedBits uint = pos & 7
		var nChangedBits uint = brotliMinSizeT(n_bits, 8-nUnchangedBits)
		var totalBits uint = nUnchangedBits + nChangedBits
		var mask uint32 = (^((1 << totalBits) - 1)) | ((1 << nUnchangedBits) - 1)
		var unchangedBits uint32 = uint32(array[bytePos]) & mask
		var changedBits uint32 = bits & ((1 << nChangedBits) - 1)
		array[bytePos] = byte(changedBits<<nUnchangedBits | unchangedBits)
		n_bits -= nChangedBits
		bits >>= nChangedBits
		pos += nChangedBits
	}
}

func rewindBitPosition1(newStorageIx uint, storageIx *uint, storage []byte) {
	var bitpos uint = newStorageIx & 7
	var mask uint = (1 << bitpos) - 1
	storage[newStorageIx>>3] &= byte(mask)
	*storageIx = newStorageIx
}

var shouldmergeblockKsamplerate uint = 43

func shouldMergeBlock(data []byte, len uint, depths []byte) bool {
	var histo = [256]uint{0}
	var i uint
	for i = 0; i < len; i += shouldmergeblockKsamplerate {
		histo[data[i]]++
	}
	{
		var total uint = (len + shouldmergeblockKsamplerate - 1) / shouldmergeblockKsamplerate
		var r float64 = (fastLog2(total)+0.5)*float64(total) + 200
		for i = 0; i < 256; i++ {
			r -= float64(histo[i]) * (float64(depths[i]) + fastLog2(histo[i]))
		}

		return r >= 0.0
	}
}

func shouldUseUncompressedMode(metablockStart []byte, nextEmit []byte, insertlen uint, literalRatio uint) bool {
	var compressed uint = uint(-cap(nextEmit) + cap(metablockStart))
	if compressed*50 > insertlen {
		return false
	} else {
		return literalRatio > 980
	}
}

func emitUncompressedMetaBlock1(begin []byte, end []byte, storageIxStart uint, storageIx *uint, storage []byte) {
	var len uint = uint(-cap(end) + cap(begin))
	rewindBitPosition1(storageIxStart, storageIx, storage)
	storeMetaBlockHeader1(uint(len), true, storageIx, storage)
	*storageIx = (*storageIx + 7) &^ 7
	copy(storage[*storageIx>>3:], begin[:len])
	*storageIx += uint(len << 3)
	storage[*storageIx>>3] = 0
}

var kCmdHistoSeed = [128]uint32{
	0,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	0,
	0,
	0,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	0,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	0,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	0,
	0,
	0,
	0,
}

var compressfragmentfastimplKfirstblocksize uint = 3 << 15
var compressfragmentfastimplKmergeblocksize uint = 1 << 16

func compressFragmentFastImpl(in []byte, inputSize uint, isLast bool, table []int, tableBits uint, cmdDepth []byte, cmdBits []uint16, cmdCodeNumbits *uint, cmdCode []byte, storageIx *uint, storage []byte) {
	var cmdHisto [128]uint32
	var ipEnd int
	var nextEmit int = 0
	var baseIp int = 0
	var input int = 0
	const kInputMarginBytes uint = windowGap
	const kMinMatchLen uint = 5
	var metablockStart int = input
	var blockSize uint = brotliMinSizeT(inputSize, compressfragmentfastimplKfirstblocksize)
	var totalBlockSize uint = blockSize
	var mlenStorageIx uint = *storageIx + 3
	var litDepth [256]byte
	var litBits [256]uint16
	var literalRatio uint
	var ip int
	var lastDistance int
	var shift uint = 64 - tableBits

	/* "next_emit" is a pointer to the first byte that is not covered by a
	   previous copy. Bytes between "next_emit" and the start of the next copy or
	   the end of the input will be emitted as literal bytes. */

	/* Save the start of the first block for position and distance computations.
	 */

	/* Save the bit position of the MLEN field of the meta-block header, so that
	   we can update it later if we decide to extend this meta-block. */
	storeMetaBlockHeader1(blockSize, false, storageIx, storage)

	/* No block splits, no contexts. */
	writeBits(13, 0, storageIx, storage)

	literalRatio = buildAndStoreLiteralPrefixCode(in[input:], blockSize, litDepth[:], litBits[:], storageIx, storage)
	{
		/* Store the pre-compressed command and distance prefix codes. */
		var i uint
		for i = 0; i+7 < *cmdCodeNumbits; i += 8 {
			writeBits(8, uint64(cmdCode[i>>3]), storageIx, storage)
		}
	}

	writeBits(*cmdCodeNumbits&7, uint64(cmdCode[*cmdCodeNumbits>>3]), storageIx, storage)

	/* Initialize the command and distance histograms. We will gather
	   statistics of command and distance codes during the processing
	   of this block and use it to update the command and distance
	   prefix codes for the next block. */
emitCommands:
	copy(cmdHisto[:], kCmdHistoSeed[:])

	/* "ip" is the input pointer. */
	ip = input

	lastDistance = -1
	ipEnd = int(uint(input) + blockSize)

	if blockSize >= kInputMarginBytes {
		var lenLimit uint = brotliMinSizeT(blockSize-kMinMatchLen, inputSize-kInputMarginBytes)
		var ipLimit int = int(uint(input) + lenLimit)
		/* For the last block, we need to keep a 16 bytes margin so that we can be
		   sure that all distances are at most window size - 16.
		   For all other blocks, we only need to keep a margin of 5 bytes so that
		   we don't go over the block size with a copy. */

		var nextHash uint32
		ip++
		for nextHash = hash5(in[ip:], shift); ; {
			var skip uint32 = 32
			var nextIp int = ip
			/* Step 1: Scan forward in the input looking for a 5-byte-long match.
			   If we get close to exhausting the input then goto emit_remainder.

			   Heuristic match skipping: If 32 bytes are scanned with no matches
			   found, start looking only at every other byte. If 32 more bytes are
			   scanned, look at every third byte, etc.. When a match is found,
			   immediately go back to looking at every byte. This is a small loss
			   (~5% performance, ~0.1% density) for compressible data due to more
			   bookkeeping, but for non-compressible data (such as JPEG) it's a huge
			   win since the compressor quickly "realizes" the data is incompressible
			   and doesn't bother looking for matches everywhere.

			   The "skip" variable keeps track of how many bytes there are since the
			   last match; dividing it by 32 (i.e. right-shifting by five) gives the
			   number of bytes to move ahead for each iteration. */

			var candidate int
			assert(nextEmit < ip)

		trawl:
			for {
				var hash uint32 = nextHash
				var bytesBetweenHashLookups uint32 = skip >> 5
				skip++
				assert(hash == hash5(in[nextIp:], shift))
				ip = nextIp
				nextIp = int(uint32(ip) + bytesBetweenHashLookups)
				if nextIp > ipLimit {
					goto emitRemainder
				}

				nextHash = hash5(in[nextIp:], shift)
				candidate = ip - lastDistance
				if isMatch5(in[ip:], in[candidate:]) {
					if candidate < ip {
						table[hash] = int(ip - baseIp)
						break
					}
				}

				candidate = baseIp + table[hash]
				assert(candidate >= baseIp)
				assert(candidate < ip)

				table[hash] = int(ip - baseIp)
				if isMatch5(in[ip:], in[candidate:]) {
					break
				}
			}

			/* Check copy distance. If candidate is not feasible, continue search.
			   Checking is done outside of hot loop to reduce overhead. */
			if ip-candidate > maxdistanceCompressFragment {
				goto trawl
			}

			/* Step 2: Emit the found match together with the literal bytes from
			   "next_emit" to the bit stream, and then see if we can find a next match
			   immediately afterwards. Repeat until we find no match for the input
			   without emitting some literal bytes. */
			{
				var base int = ip
				/* > 0 */
				var matched uint = 5 + findMatchLengthWithLimit(in[candidate+5:], in[ip+5:], uint(ipEnd-ip)-5)
				var distance int = int(base - candidate)
				/* We have a 5-byte match at ip, and we need to emit bytes in
				   [next_emit, ip). */

				var insert uint = uint(base - nextEmit)
				ip += int(matched)
				if insert < 6210 {
					emitInsertLen1(insert, cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)
				} else if shouldUseUncompressedMode(in[metablockStart:], in[nextEmit:], insert, literalRatio) {
					emitUncompressedMetaBlock1(in[metablockStart:], in[base:], mlenStorageIx-3, storageIx, storage)
					inputSize -= uint(base - input)
					input = base
					nextEmit = input
					goto nextBlock
				} else {
					emitLongInsertLen(insert, cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)
				}

				emitLiterals(in[nextEmit:], insert, litDepth[:], litBits[:], storageIx, storage)
				if distance == lastDistance {
					writeBits(uint(cmdDepth[64]), uint64(cmdBits[64]), storageIx, storage)
					cmdHisto[64]++
				} else {
					emitDistance1(uint(distance), cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)
					lastDistance = distance
				}

				emitCopyLenLastDistance1(matched, cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)

				nextEmit = ip
				if ip >= ipLimit {
					goto emitRemainder
				}

				/* We could immediately start working at ip now, but to improve
				   compression we first update "table" with the hashes of some positions
				   within the last copy. */
				{
					var inputBytes uint64 = binary.LittleEndian.Uint64(in[ip-3:])
					var prevHash uint32 = hashBytesAtOffset5(inputBytes, 0, shift)
					var curHash uint32 = hashBytesAtOffset5(inputBytes, 3, shift)
					table[prevHash] = int(ip - baseIp - 3)
					prevHash = hashBytesAtOffset5(inputBytes, 1, shift)
					table[prevHash] = int(ip - baseIp - 2)
					prevHash = hashBytesAtOffset5(inputBytes, 2, shift)
					table[prevHash] = int(ip - baseIp - 1)

					candidate = baseIp + table[curHash]
					table[curHash] = int(ip - baseIp)
				}
			}

			for isMatch5(in[ip:], in[candidate:]) {
				var base int = ip
				/* We have a 5-byte match at ip, and no need to emit any literal bytes
				   prior to ip. */

				var matched uint = 5 + findMatchLengthWithLimit(in[candidate+5:], in[ip+5:], uint(ipEnd-ip)-5)
				if ip-candidate > maxdistanceCompressFragment {
					break
				}
				ip += int(matched)
				lastDistance = int(base - candidate) /* > 0 */
				emitCopyLen1(matched, cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)
				emitDistance1(uint(lastDistance), cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)

				nextEmit = ip
				if ip >= ipLimit {
					goto emitRemainder
				}

				/* We could immediately start working at ip now, but to improve
				   compression we first update "table" with the hashes of some positions
				   within the last copy. */
				{
					var inputBytes uint64 = binary.LittleEndian.Uint64(in[ip-3:])
					var prevHash uint32 = hashBytesAtOffset5(inputBytes, 0, shift)
					var curHash uint32 = hashBytesAtOffset5(inputBytes, 3, shift)
					table[prevHash] = int(ip - baseIp - 3)
					prevHash = hashBytesAtOffset5(inputBytes, 1, shift)
					table[prevHash] = int(ip - baseIp - 2)
					prevHash = hashBytesAtOffset5(inputBytes, 2, shift)
					table[prevHash] = int(ip - baseIp - 1)

					candidate = baseIp + table[curHash]
					table[curHash] = int(ip - baseIp)
				}
			}

			ip++
			nextHash = hash5(in[ip:], shift)
		}
	}

emitRemainder:
	assert(nextEmit <= ipEnd)
	input += int(blockSize)
	inputSize -= blockSize
	blockSize = brotliMinSizeT(inputSize, compressfragmentfastimplKmergeblocksize)

	/* Decide if we want to continue this meta-block instead of emitting the
	   last insert-only command. */
	if inputSize > 0 && totalBlockSize+blockSize <= 1<<20 && shouldMergeBlock(in[input:], blockSize, litDepth[:]) {
		assert(totalBlockSize > 1<<16)

		/* Update the size of the current meta-block and continue emitting commands.
		   We can do this because the current size and the new size both have 5
		   nibbles. */
		totalBlockSize += blockSize

		updateBits(20, uint32(totalBlockSize-1), mlenStorageIx, storage)
		goto emitCommands
	}

	/* Emit the remaining bytes as literals. */
	if nextEmit < ipEnd {
		var insert uint = uint(ipEnd - nextEmit)
		if insert < 6210 {
			emitInsertLen1(insert, cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)
			emitLiterals(in[nextEmit:], insert, litDepth[:], litBits[:], storageIx, storage)
		} else if shouldUseUncompressedMode(in[metablockStart:], in[nextEmit:], insert, literalRatio) {
			emitUncompressedMetaBlock1(in[metablockStart:], in[ipEnd:], mlenStorageIx-3, storageIx, storage)
		} else {
			emitLongInsertLen(insert, cmdDepth, cmdBits, cmdHisto[:], storageIx, storage)
			emitLiterals(in[nextEmit:], insert, litDepth[:], litBits[:], storageIx, storage)
		}
	}

	nextEmit = ipEnd

	/* If we have more data, write a new meta-block header and prefix codes and
	   then continue emitting commands. */
nextBlock:
	if inputSize > 0 {
		metablockStart = input
		blockSize = brotliMinSizeT(inputSize, compressfragmentfastimplKfirstblocksize)
		totalBlockSize = blockSize

		/* Save the bit position of the MLEN field of the meta-block header, so that
		   we can update it later if we decide to extend this meta-block. */
		mlenStorageIx = *storageIx + 3

		storeMetaBlockHeader1(blockSize, false, storageIx, storage)

		/* No block splits, no contexts. */
		writeBits(13, 0, storageIx, storage)

		literalRatio = buildAndStoreLiteralPrefixCode(in[input:], blockSize, litDepth[:], litBits[:], storageIx, storage)
		buildAndStoreCommandPrefixCode1(cmdHisto[:], cmdDepth, cmdBits, storageIx, storage)
		goto emitCommands
	}

	if !isLast {
		/* If this is not the last block, update the command and distance prefix
		   codes for the next block and store the compressed forms. */
		cmdCode[0] = 0

		*cmdCodeNumbits = 0
		buildAndStoreCommandPrefixCode1(cmdHisto[:], cmdDepth, cmdBits, cmdCodeNumbits, cmdCode)
	}
}

/*
Compresses "input" string to the "*storage" buffer as one or more complete

	meta-blocks, and updates the "*storage_ix" bit position.

	If "is_last" is 1, emits an additional empty last meta-block.

	"cmd_depth" and "cmd_bits" contain the command and distance prefix codes
	(see comment in encode.h) used for the encoding of this input fragment.
	If "is_last" is 0, they are updated to reflect the statistics
	of this input fragment, to be used for the encoding of the next fragment.

	"*cmd_code_numbits" is the number of bits of the compressed representation
	of the command and distance prefix codes, and "cmd_code" is an array of
	at least "(*cmd_code_numbits + 7) >> 3" size that contains the compressed
	command and distance prefix codes. If "is_last" is 0, these are also
	updated to represent the updated "cmd_depth" and "cmd_bits".

	REQUIRES: "input_size" is greater than zero, or "is_last" is 1.
	REQUIRES: "input_size" is less or equal to maximal metablock size (1 << 24).
	REQUIRES: All elements in "table[0..table_size-1]" are initialized to zero.
	REQUIRES: "table_size" is an odd (9, 11, 13, 15) power of two
	OUTPUT: maximal copy distance <= |input_size|
	OUTPUT: maximal copy distance <= BROTLI_MAX_BACKWARD_LIMIT(18)
*/
func compressFragmentFast(input []byte, inputSize uint, isLast bool, table []int, tableSize uint, cmdDepth []byte, cmdBits []uint16, cmdCodeNumbits *uint, cmdCode []byte, storageIx *uint, storage []byte) {
	var initialStorageIx uint = *storageIx
	var tableBits uint = uint(log2FloorNonZero(tableSize))

	if inputSize == 0 {
		assert(isLast)
		writeBits(1, 1, storageIx, storage) /* islast */
		writeBits(1, 1, storageIx, storage) /* isempty */
		*storageIx = (*storageIx + 7) &^ 7
		return
	}

	compressFragmentFastImpl(input, inputSize, isLast, table, tableBits, cmdDepth, cmdBits, cmdCodeNumbits, cmdCode, storageIx, storage)

	/* If output is larger than single uncompressed block, rewrite it. */
	if *storageIx-initialStorageIx > 31+(inputSize<<3) {
		emitUncompressedMetaBlock1(input, input[inputSize:], initialStorageIx, storageIx, storage)
	}

	if isLast {
		writeBits(1, 1, storageIx, storage) /* islast */
		writeBits(1, 1, storageIx, storage) /* isempty */
		*storageIx = (*storageIx + 7) &^ 7
	}
}
