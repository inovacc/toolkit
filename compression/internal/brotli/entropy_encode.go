package brotli

import "math"

/* Copyright 2010 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Entropy encoding (Huffman) utilities. */

/* A node of a Huffman tree. */
type huffmanTree struct {
	totalCount        uint32
	indexLeft         int16
	indexRightOrValue int16
}

func initHuffmanTree(self *huffmanTree, count uint32, left int16, right int16) {
	self.totalCount = count
	self.indexLeft = left
	self.indexRightOrValue = right
}

/* Input size optimized Shell sort. */
type huffmanTreeComparator func(huffmanTree, huffmanTree) bool

var sorthuffmantreeitemsGaps = []uint{132, 57, 23, 10, 4, 1}

func sortHuffmanTreeItems(items []huffmanTree, n uint, comparator huffmanTreeComparator) {
	if n < 13 {
		/* Insertion sort. */
		var i uint
		for i = 1; i < n; i++ {
			var tmp huffmanTree = items[i]
			var k uint = i
			var j uint = i - 1
			for comparator(tmp, items[j]) {
				items[k] = items[j]
				k = j
				if j == 0 {
					break
				}
				j--
			}

			items[k] = tmp
		}

		return
	} else {
		var g int
		if n < 57 {
			g = 2
		} else {
			g = 0
		}
		for ; g < 6; g++ {
			var gap uint = sorthuffmantreeitemsGaps[g]
			var i uint
			for i = gap; i < n; i++ {
				var j uint = i
				var tmp huffmanTree = items[i]
				for ; j >= gap && comparator(tmp, items[j-gap]); j -= gap {
					items[j] = items[j-gap]
				}

				items[j] = tmp
			}
		}
	}
}

/* Returns 1 if assignment of depths succeeded, otherwise 0. */
func setDepth(p0 int, pool []huffmanTree, depth []byte, maxDepth int) bool {
	var stack [16]int
	var level int = 0
	var p int = p0
	assert(maxDepth <= 15)
	stack[0] = -1
	for {
		if pool[p].indexLeft >= 0 {
			level++
			if level > maxDepth {
				return false
			}
			stack[level] = int(pool[p].indexRightOrValue)
			p = int(pool[p].indexLeft)
			continue
		} else {
			depth[pool[p].indexRightOrValue] = byte(level)
		}

		for level >= 0 && stack[level] == -1 {
			level--
		}
		if level < 0 {
			return true
		}
		p = stack[level]
		stack[level] = -1
	}
}

/* Sort the root nodes, least popular first. */
func sortHuffmanTree(v0 huffmanTree, v1 huffmanTree) bool {
	if v0.totalCount != v1.totalCount {
		return v0.totalCount < v1.totalCount
	}

	return v0.indexRightOrValue > v1.indexRightOrValue
}

/*
This function will create a Huffman tree.

	The catch here is that the tree cannot be arbitrarily deep.
	Brotli specifies a maximum depth of 15 bits for "code trees"
	and 7 bits for "code length code trees."

	count_limit is the value that is to be faked as the minimum value
	and this minimum value is raised until the tree matches the
	maximum length requirement.

	This algorithm is not of excellent performance for very long data blocks,
	especially when population counts are longer than 2**tree_limit, but
	we are not planning to use this with extremely long blocks.

	See http://en.wikipedia.org/wiki/Huffman_coding
*/
func createHuffmanTree(data []uint32, length uint, treeLimit int, tree []huffmanTree, depth []byte) {
	var countLimit uint32
	var sentinel huffmanTree
	initHuffmanTree(&sentinel, math.MaxUint32, -1, -1)

	/* For block sizes below 64 kB, we never need to do a second iteration
	   of this loop. Probably all of our block sizes will be smaller than
	   that, so this loop is mostly of academic interest. If we actually
	   would need this, we would be better off with the Katajainen algorithm. */
	for countLimit = 1; ; countLimit *= 2 {
		var n uint = 0
		var i uint
		var j uint
		var k uint
		for i = length; i != 0; {
			i--
			if data[i] != 0 {
				var count uint32 = brotliMaxUint32T(data[i], countLimit)
				initHuffmanTree(&tree[n], count, -1, int16(i))
				n++
			}
		}

		if n == 1 {
			depth[tree[0].indexRightOrValue] = 1 /* Only one element. */
			break
		}

		sortHuffmanTreeItems(tree, n, huffmanTreeComparator(sortHuffmanTree))

		/* The nodes are:
		   [0, n): the sorted leaf nodes that we start with.
		   [n]: we add a sentinel here.
		   [n + 1, 2n): new parent nodes are added here, starting from
		                (n+1). These are naturally in ascending order.
		   [2n]: we add a sentinel at the end as well.
		   There will be (2n+1) elements at the end. */
		tree[n] = sentinel

		tree[n+1] = sentinel

		i = 0     /* Points to the next leaf node. */
		j = n + 1 /* Points to the next non-leaf node. */
		for k = n - 1; k != 0; k-- {
			var left uint
			var right uint
			if tree[i].totalCount <= tree[j].totalCount {
				left = i
				i++
			} else {
				left = j
				j++
			}

			if tree[i].totalCount <= tree[j].totalCount {
				right = i
				i++
			} else {
				right = j
				j++
			}
			{
				/* The sentinel node becomes the parent node. */
				var jEnd uint = 2*n - k
				tree[jEnd].totalCount = tree[left].totalCount + tree[right].totalCount
				tree[jEnd].indexLeft = int16(left)
				tree[jEnd].indexRightOrValue = int16(right)

				/* Add back the last sentinel node. */
				tree[jEnd+1] = sentinel
			}
		}

		if setDepth(int(2*n-1), tree[0:], depth, treeLimit) {
			/* We need to pack the Huffman tree in tree_limit bits. If this was not
			   successful, add fake entities to the lowest values and retry. */
			break
		}
	}
}

func reverse(v []byte, start uint, end uint) {
	end--
	for start < end {
		var tmp byte = v[start]
		v[start] = v[end]
		v[end] = tmp
		start++
		end--
	}
}

func writeHuffmanTreeRepetitions(previousValue byte, value byte, repetitions uint, treeSize *uint, tree []byte, extraBitsData []byte) {
	assert(repetitions > 0)
	if previousValue != value {
		tree[*treeSize] = value
		extraBitsData[*treeSize] = 0
		*treeSize++
		repetitions--
	}

	if repetitions == 7 {
		tree[*treeSize] = value
		extraBitsData[*treeSize] = 0
		*treeSize++
		repetitions--
	}

	if repetitions < 3 {
		var i uint
		for i = 0; i < repetitions; i++ {
			tree[*treeSize] = value
			extraBitsData[*treeSize] = 0
			*treeSize++
		}
	} else {
		var start uint = *treeSize
		repetitions -= 3
		for {
			tree[*treeSize] = repeatPreviousCodeLength
			extraBitsData[*treeSize] = byte(repetitions & 0x3)
			*treeSize++
			repetitions >>= 2
			if repetitions == 0 {
				break
			}

			repetitions--
		}

		reverse(tree, start, *treeSize)
		reverse(extraBitsData, start, *treeSize)
	}
}

func writeHuffmanTreeRepetitionsZeros(repetitions uint, treeSize *uint, tree []byte, extraBitsData []byte) {
	if repetitions == 11 {
		tree[*treeSize] = 0
		extraBitsData[*treeSize] = 0
		*treeSize++
		repetitions--
	}

	if repetitions < 3 {
		var i uint
		for i = 0; i < repetitions; i++ {
			tree[*treeSize] = 0
			extraBitsData[*treeSize] = 0
			*treeSize++
		}
	} else {
		var start uint = *treeSize
		repetitions -= 3
		for {
			tree[*treeSize] = repeatZeroCodeLength
			extraBitsData[*treeSize] = byte(repetitions & 0x7)
			*treeSize++
			repetitions >>= 3
			if repetitions == 0 {
				break
			}

			repetitions--
		}

		reverse(tree, start, *treeSize)
		reverse(extraBitsData, start, *treeSize)
	}
}

/*
Change the population counts in a way that the consequent

	Huffman tree compression, especially its RLE-part will be more
	likely to compress this data more efficiently.

	length contains the size of the histogram.
	counts contains the population counts.
	good_for_rle is a buffer of at least length size
*/
func optimizeHuffmanCountsForRLE(length uint, counts []uint32, goodForRle []byte) {
	var nonzeroCount uint = 0
	var stride uint
	var limit uint
	var sum uint
	var streakLimit uint = 1240
	var i uint
	/* Let's make the Huffman code more compatible with RLE encoding. */
	for i = 0; i < length; i++ {
		if counts[i] != 0 {
			nonzeroCount++
		}
	}

	if nonzeroCount < 16 {
		return
	}

	for length != 0 && counts[length-1] == 0 {
		length--
	}

	if length == 0 {
		return /* All zeros. */
	}

	/* Now counts[0..length - 1] does not have trailing zeros. */
	{
		var nonzeros uint = 0
		var smallestNonzero uint32 = 1 << 30
		for i = 0; i < length; i++ {
			if counts[i] != 0 {
				nonzeros++
				if smallestNonzero > counts[i] {
					smallestNonzero = counts[i]
				}
			}
		}

		if nonzeros < 5 {
			/* Small histogram will model it well. */
			return
		}

		if smallestNonzero < 4 {
			var zeros uint = length - nonzeros
			if zeros < 6 {
				for i = 1; i < length-1; i++ {
					if counts[i-1] != 0 && counts[i] == 0 && counts[i+1] != 0 {
						counts[i] = 1
					}
				}
			}
		}

		if nonzeros < 28 {
			return
		}
	}

	/* 2) Let's mark all population counts that already can be encoded
	   with an RLE code. */
	for i := 0; i < int(length); i++ {
		goodForRle[i] = 0
	}
	{
		var symbol uint32 = counts[0]
		/* Let's not spoil any of the existing good RLE codes.
		   Mark any seq of 0's that is longer as 5 as a good_for_rle.
		   Mark any seq of non-0's that is longer as 7 as a good_for_rle. */

		var step uint = 0
		for i = 0; i <= length; i++ {
			if i == length || counts[i] != symbol {
				if (symbol == 0 && step >= 5) || (symbol != 0 && step >= 7) {
					var k uint
					for k = 0; k < step; k++ {
						goodForRle[i-k-1] = 1
					}
				}

				step = 1
				if i != length {
					symbol = counts[i]
				}
			} else {
				step++
			}
		}
	}

	/* 3) Let's replace those population counts that lead to more RLE codes.
	   Math here is in 24.8 fixed point representation. */
	stride = 0

	limit = uint(256*(counts[0]+counts[1]+counts[2])/3 + 420)
	sum = 0
	for i = 0; i <= length; i++ {
		if i == length || goodForRle[i] != 0 || (i != 0 && goodForRle[i-1] != 0) || (256*counts[i]-uint32(limit)+uint32(streakLimit)) >= uint32(2*streakLimit) {
			if stride >= 4 || (stride >= 3 && sum == 0) {
				var k uint
				var count uint = (sum + stride/2) / stride
				/* The stride must end, collapse what we have, if we have enough (4). */
				if count == 0 {
					count = 1
				}

				if sum == 0 {
					/* Don't make an all zeros stride to be upgraded to ones. */
					count = 0
				}

				for k = 0; k < stride; k++ {
					/* We don't want to change value at counts[i],
					   that is already belonging to the next stride. Thus - 1. */
					counts[i-k-1] = uint32(count)
				}
			}

			stride = 0
			sum = 0
			if i < length-2 {
				/* All interesting strides have a count of at least 4, */
				/* at least when non-zeros. */
				limit = uint(256*(counts[i]+counts[i+1]+counts[i+2])/3 + 420)
			} else if i < length {
				limit = uint(256 * counts[i])
			} else {
				limit = 0
			}
		}

		stride++
		if i != length {
			sum += uint(counts[i])
			if stride >= 4 {
				limit = (256*sum + stride/2) / stride
			}

			if stride == 4 {
				limit += 120
			}
		}
	}
}

func decideOverRLEUse(depth []byte, length uint, useRleForNonZero *bool, useRleForZero *bool) {
	var totalRepsZero uint = 0
	var totalRepsNonZero uint = 0
	var countRepsZero uint = 1
	var countRepsNonZero uint = 1
	var i uint
	for i = 0; i < length; {
		var value byte = depth[i]
		var reps uint = 1
		var k uint
		for k = i + 1; k < length && depth[k] == value; k++ {
			reps++
		}

		if reps >= 3 && value == 0 {
			totalRepsZero += reps
			countRepsZero++
		}

		if reps >= 4 && value != 0 {
			totalRepsNonZero += reps
			countRepsNonZero++
		}

		i += reps
	}

	*useRleForNonZero = totalRepsNonZero > countRepsNonZero*2
	*useRleForZero = totalRepsZero > countRepsZero*2
}

/*
Write a Huffman tree from bit depths into the bit-stream representation

	of a Huffman tree. The generated Huffman tree is to be compressed once
	more using a Huffman tree
*/
func writeHuffmanTree(depth []byte, length uint, treeSize *uint, tree []byte, extraBitsData []byte) {
	var previousValue byte = initialRepeatedCodeLength
	var i uint
	var useRleForNonZero bool = false
	var useRleForZero bool = false
	var newLength uint = length
	/* Throw away trailing zeros. */
	for i = 0; i < length; i++ {
		if depth[length-i-1] == 0 {
			newLength--
		} else {
			break
		}
	}

	/* First gather statistics on if it is a good idea to do RLE. */
	if length > 50 {
		/* Find RLE coding for longer codes.
		   Shorter codes seem not to benefit from RLE. */
		decideOverRLEUse(depth, newLength, &useRleForNonZero, &useRleForZero)
	}

	/* Actual RLE coding. */
	for i = 0; i < newLength; {
		var value byte = depth[i]
		var reps uint = 1
		if (value != 0 && useRleForNonZero) || (value == 0 && useRleForZero) {
			var k uint
			for k = i + 1; k < newLength && depth[k] == value; k++ {
				reps++
			}
		}

		if value == 0 {
			writeHuffmanTreeRepetitionsZeros(reps, treeSize, tree, extraBitsData)
		} else {
			writeHuffmanTreeRepetitions(previousValue, value, reps, treeSize, tree, extraBitsData)
			previousValue = value
		}

		i += reps
	}
}

var reversebitsKlut = [16]uint{
	0x00,
	0x08,
	0x04,
	0x0C,
	0x02,
	0x0A,
	0x06,
	0x0E,
	0x01,
	0x09,
	0x05,
	0x0D,
	0x03,
	0x0B,
	0x07,
	0x0F,
}

func reverseBits(numBits uint, bits uint16) uint16 {
	var retval uint = reversebitsKlut[bits&0x0F]
	var i uint
	for i = 4; i < numBits; i += 4 {
		retval <<= 4
		bits = uint16(bits >> 4)
		retval |= reversebitsKlut[bits&0x0F]
	}

	retval >>= (0 - numBits) & 0x03
	return uint16(retval)
}

/* 0..15 are values for bits */
const maxHuffmanBits = 16

/* Get the actual bit values for a tree of bit depths. */
func convertBitDepthsToSymbols(depth []byte, len uint, bits []uint16) {
	var blCount = [maxHuffmanBits]uint16{0}
	var nextCode [maxHuffmanBits]uint16
	var i uint
	/* In Brotli, all bit depths are [1..15]
	   0 bit depth means that the symbol does not exist. */

	var code int = 0
	for i = 0; i < len; i++ {
		blCount[depth[i]]++
	}

	blCount[0] = 0
	nextCode[0] = 0
	for i = 1; i < maxHuffmanBits; i++ {
		code = (code + int(blCount[i-1])) << 1
		nextCode[i] = uint16(code)
	}

	for i = 0; i < len; i++ {
		if depth[i] != 0 {
			bits[i] = reverseBits(uint(depth[i]), nextCode[depth[i]])
			nextCode[depth[i]]++
		}
	}
}
