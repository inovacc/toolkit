package brotli

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Block split point selection utilities. */

type blockSplit struct {
	numTypes         uint
	numBlocks        uint
	types            []byte
	lengths          []uint32
	typesAllocSize   uint
	lengthsAllocSize uint
}

const (
	kMaxLiteralHistograms        uint    = 100
	kMaxCommandHistograms        uint    = 50
	kLiteralBlockSwitchCost      float64 = 28.1
	kCommandBlockSwitchCost      float64 = 13.5
	kDistanceBlockSwitchCost     float64 = 14.6
	kLiteralStrideLength         uint    = 70
	kCommandStrideLength         uint    = 40
	kSymbolsPerLiteralHistogram  uint    = 544
	kSymbolsPerCommandHistogram  uint    = 530
	kSymbolsPerDistanceHistogram uint    = 544
	kMinLengthForBlockSplitting  uint    = 128
	kIterMulForRefining          uint    = 2
	kMinItersForRefining         uint    = 100
)

func countLiterals(cmds []command) uint {
	var totalLength uint = 0
	/* Count how many we have. */

	for i := range cmds {
		totalLength += uint(cmds[i].insert_len_)
	}

	return totalLength
}

func copyLiteralsToByteArray(cmds []command, data []byte, offset uint, mask uint, literals []byte) {
	var pos uint = 0
	var fromPos uint = offset & mask
	for i := range cmds {
		var insertLen uint = uint(cmds[i].insert_len_)
		if fromPos+insertLen > mask {
			var headSize uint = mask + 1 - fromPos
			copy(literals[pos:], data[fromPos:][:headSize])
			fromPos = 0
			pos += headSize
			insertLen -= headSize
		}

		if insertLen > 0 {
			copy(literals[pos:], data[fromPos:][:insertLen])
			pos += insertLen
		}

		fromPos = uint((uint32(fromPos+insertLen) + commandCopyLen(&cmds[i])) & uint32(mask))
	}
}

func myRand(seed *uint32) uint32 {
	/* Initial seed should be 7. In this case, loop length is (1 << 29). */
	*seed *= 16807

	return *seed
}

func bitCost(count uint) float64 {
	if count == 0 {
		return -2.0
	} else {
		return fastLog2(count)
	}
}

const histogramsPerBatch = 64

const clustersPerBatch = 16

func initBlockSplit(self *blockSplit) {
	self.numTypes = 0
	self.numBlocks = 0
	self.types = self.types[:0]
	self.lengths = self.lengths[:0]
	self.typesAllocSize = 0
	self.lengthsAllocSize = 0
}

func splitBlock(cmds []command, data []byte, pos uint, mask uint, params *encoderParams, literalSplit *blockSplit, insertAndCopySplit *blockSplit, distSplit *blockSplit) {
	{
		var literalsCount uint = countLiterals(cmds)
		var literals []byte = make([]byte, literalsCount)

		/* Create a continuous array of literals. */
		copyLiteralsToByteArray(cmds, data, pos, mask, literals)

		/* Create the block split on the array of literals.
		   Literal histograms have alphabet size 256. */
		splitByteVectorLiteral(literals, literalsCount, kSymbolsPerLiteralHistogram, kMaxLiteralHistograms, kLiteralStrideLength, kLiteralBlockSwitchCost, params, literalSplit)

		literals = nil
	}
	{
		var insertAndCopyCodes []uint16 = make([]uint16, len(cmds))
		/* Compute prefix codes for commands. */

		for i := range cmds {
			insertAndCopyCodes[i] = cmds[i].cmd_prefix_
		}

		/* Create the block split on the array of command prefixes. */
		splitByteVectorCommand(insertAndCopyCodes, kSymbolsPerCommandHistogram, kMaxCommandHistograms, kCommandStrideLength, kCommandBlockSwitchCost, params, insertAndCopySplit)

		/* TODO: reuse for distances? */

		insertAndCopyCodes = nil
	}
	{
		var distancePrefixes []uint16 = make([]uint16, len(cmds))
		var j uint = 0
		/* Create a continuous array of distance prefixes. */

		for i := range cmds {
			var cmd *command = &cmds[i]
			if commandCopyLen(cmd) != 0 && cmd.cmd_prefix_ >= 128 {
				distancePrefixes[j] = cmd.dist_prefix_ & 0x3FF
				j++
			}
		}

		/* Create the block split on the array of distance prefixes. */
		splitByteVectorDistance(distancePrefixes, j, kSymbolsPerDistanceHistogram, kMaxCommandHistograms, kCommandStrideLength, kDistanceBlockSwitchCost, params, distSplit)

		distancePrefixes = nil
	}
}
