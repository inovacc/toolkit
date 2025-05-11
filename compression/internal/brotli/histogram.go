package brotli

import "math"

/* The distance symbols effectively used by "Large Window Brotli" (32-bit). */
const numHistogramDistanceSymbols = 544

type histogramLiteral struct {
	data_      [numLiteralSymbols]uint32
	totalCount uint
	bitCost    float64
}

func histogramClearLiteral(self *histogramLiteral) {
	self.data_ = [numLiteralSymbols]uint32{}
	self.totalCount = 0
	self.bitCost = math.MaxFloat64
}

func clearHistogramsLiteral(array []histogramLiteral, length uint) {
	var i uint
	for i = 0; i < length; i++ {
		histogramClearLiteral(&array[i:][0])
	}
}

func histogramAddLiteral(self *histogramLiteral, val uint) {
	self.data_[val]++
	self.totalCount++
}

func histogramAddVectorLiteral(self *histogramLiteral, p []byte, n uint) {
	self.totalCount += n
	n += 1
	for {
		n--
		if n == 0 {
			break
		}
		self.data_[p[0]]++
		p = p[1:]
	}
}

func histogramAddHistogramLiteral(self *histogramLiteral, v *histogramLiteral) {
	var i uint
	self.totalCount += v.totalCount
	for i = 0; i < numLiteralSymbols; i++ {
		self.data_[i] += v.data_[i]
	}
}

func histogramDataSizeLiteral() uint {
	return numLiteralSymbols
}

type histogramCommand struct {
	data_      [numCommandSymbols]uint32
	totalCount uint
	bitCost    float64
}

func histogramClearCommand(self *histogramCommand) {
	self.data_ = [numCommandSymbols]uint32{}
	self.totalCount = 0
	self.bitCost = math.MaxFloat64
}

func clearHistogramsCommand(array []histogramCommand, length uint) {
	var i uint
	for i = 0; i < length; i++ {
		histogramClearCommand(&array[i:][0])
	}
}

func histogramAddCommand(self *histogramCommand, val uint) {
	self.data_[val]++
	self.totalCount++
}

func histogramAddVectorCommand(self *histogramCommand, p []uint16, n uint) {
	self.totalCount += n
	n += 1
	for {
		n--
		if n == 0 {
			break
		}
		self.data_[p[0]]++
		p = p[1:]
	}
}

func histogramAddHistogramCommand(self *histogramCommand, v *histogramCommand) {
	var i uint
	self.totalCount += v.totalCount
	for i = 0; i < numCommandSymbols; i++ {
		self.data_[i] += v.data_[i]
	}
}

func histogramDataSizeCommand() uint {
	return numCommandSymbols
}

type histogramDistance struct {
	data_      [numDistanceSymbols]uint32
	totalCount uint
	bitCost    float64
}

func histogramClearDistance(self *histogramDistance) {
	self.data_ = [numDistanceSymbols]uint32{}
	self.totalCount = 0
	self.bitCost = math.MaxFloat64
}

func clearHistogramsDistance(array []histogramDistance, length uint) {
	var i uint
	for i = 0; i < length; i++ {
		histogramClearDistance(&array[i:][0])
	}
}

func histogramAddDistance(self *histogramDistance, val uint) {
	self.data_[val]++
	self.totalCount++
}

func histogramAddVectorDistance(self *histogramDistance, p []uint16, n uint) {
	self.totalCount += n
	n += 1
	for {
		n--
		if n == 0 {
			break
		}
		self.data_[p[0]]++
		p = p[1:]
	}
}

func histogramAddHistogramDistance(self *histogramDistance, v *histogramDistance) {
	var i uint
	self.totalCount += v.totalCount
	for i = 0; i < numDistanceSymbols; i++ {
		self.data_[i] += v.data_[i]
	}
}

func histogramDataSizeDistance() uint {
	return numDistanceSymbols
}

type blockSplitIterator struct {
	split_  *blockSplit
	idx_    uint
	type_   uint
	length_ uint
}

func initBlockSplitIterator(self *blockSplitIterator, split *blockSplit) {
	self.split_ = split
	self.idx_ = 0
	self.type_ = 0
	if len(split.lengths) > 0 {
		self.length_ = uint(split.lengths[0])
	} else {
		self.length_ = 0
	}
}

func blockSplitIteratorNext(self *blockSplitIterator) {
	if self.length_ == 0 {
		self.idx_++
		self.type_ = uint(self.split_.types[self.idx_])
		self.length_ = uint(self.split_.lengths[self.idx_])
	}

	self.length_--
}

func buildHistogramsWithContext(cmds []command, literalSplit *blockSplit, insertAndCopySplit *blockSplit, distSplit *blockSplit, ringbuffer []byte, startPos uint, mask uint, prevByte byte, prevByte2 byte, contextModes []int, literalHistograms []histogramLiteral, insertAndCopyHistograms []histogramCommand, copyDistHistograms []histogramDistance) {
	var pos uint = startPos
	var literalIt blockSplitIterator
	var insertAndCopyIt blockSplitIterator
	var distIt blockSplitIterator

	initBlockSplitIterator(&literalIt, literalSplit)
	initBlockSplitIterator(&insertAndCopyIt, insertAndCopySplit)
	initBlockSplitIterator(&distIt, distSplit)
	for i := range cmds {
		var cmd *command = &cmds[i]
		var j uint
		blockSplitIteratorNext(&insertAndCopyIt)
		histogramAddCommand(&insertAndCopyHistograms[insertAndCopyIt.type_], uint(cmd.cmd_prefix_))

		/* TODO: unwrap iterator blocks. */
		for j = uint(cmd.insert_len_); j != 0; j-- {
			var context uint
			blockSplitIteratorNext(&literalIt)
			context = literalIt.type_
			if contextModes != nil {
				var lut contextLUT = getContextLUT(contextModes[context])
				context = (context << literalContextBits) + uint(getContext(prevByte, prevByte2, lut))
			}

			histogramAddLiteral(&literalHistograms[context], uint(ringbuffer[pos&mask]))
			prevByte2 = prevByte
			prevByte = ringbuffer[pos&mask]
			pos++
		}

		pos += uint(commandCopyLen(cmd))
		if commandCopyLen(cmd) != 0 {
			prevByte2 = ringbuffer[(pos-2)&mask]
			prevByte = ringbuffer[(pos-1)&mask]
			if cmd.cmd_prefix_ >= 128 {
				var context uint
				blockSplitIteratorNext(&distIt)
				context = uint(uint32(distIt.type_<<distanceContextBits) + commandDistanceContext(cmd))
				histogramAddDistance(&copyDistHistograms[context], uint(cmd.dist_prefix_)&0x3FF)
			}
		}
	}
}
