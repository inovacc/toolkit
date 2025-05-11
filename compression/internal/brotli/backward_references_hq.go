package brotli

import "math"

type zopfliNode struct {
	length            uint32
	distance          uint32
	dcodeInsertLength uint32
	u                 struct {
		cost     float32
		next     uint32
		shortcut uint32
	}
}

const maxEffectiveDistanceAlphabetSize = 544

const kInfinity float32 = 1.7e38 /* ~= 2 ^ 127 */

var kDistanceCacheIndex = []uint32{0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1}

var kDistanceCacheOffset = []int{0, 0, 0, 0, -1, 1, -2, 2, -3, 3, -1, 1, -2, 2, -3, 3}

func initZopfliNodes(array []zopfliNode, length uint) {
	var stub zopfliNode
	var i uint
	stub.length = 1
	stub.distance = 0
	stub.dcodeInsertLength = 0
	stub.u.cost = kInfinity
	for i = 0; i < length; i++ {
		array[i] = stub
	}
}

func zopfliNodeCopyLength(self *zopfliNode) uint32 {
	return self.length & 0x1FFFFFF
}

func zopfliNodeLengthCode(self *zopfliNode) uint32 {
	var modifier uint32 = self.length >> 25
	return zopfliNodeCopyLength(self) + 9 - modifier
}

func zopfliNodeCopyDistance(self *zopfliNode) uint32 {
	return self.distance
}

func zopfliNodeDistanceCode(self *zopfliNode) uint32 {
	var shortCode uint32 = self.dcodeInsertLength >> 27
	if shortCode == 0 {
		return zopfliNodeCopyDistance(self) + numDistanceShortCodes - 1
	} else {
		return shortCode - 1
	}
}

func zopfliNodeCommandLength(self *zopfliNode) uint32 {
	return zopfliNodeCopyLength(self) + (self.dcodeInsertLength & 0x7FFFFFF)
}

/* Histogram-based cost model for zopflification. */
type zopfliCostModel struct {
	costCmd               [numCommandSymbols]float32
	costDist              []float32
	distanceHistogramSize uint32
	literalCosts          []float32
	minCostCmd            float32
	numBytes              uint
}

func initZopfliCostModel(self *zopfliCostModel, dist *distanceParams, numBytes uint) {
	var distanceHistogramSize uint32 = dist.alphabetSize
	if distanceHistogramSize > maxEffectiveDistanceAlphabetSize {
		distanceHistogramSize = maxEffectiveDistanceAlphabetSize
	}

	self.numBytes = numBytes
	self.literalCosts = make([]float32, numBytes+2)
	self.costDist = make([]float32, dist.alphabetSize)
	self.distanceHistogramSize = distanceHistogramSize
}

func cleanupZopfliCostModel(self *zopfliCostModel) {
	self.literalCosts = nil
	self.costDist = nil
}

func setCost(histogram []uint32, histogramSize uint, literalHistogram bool, cost []float32) {
	var sum uint = 0
	var missingSymbolSum uint
	var log2sum float32
	var missingSymbolCost float32
	var i uint
	for i = 0; i < histogramSize; i++ {
		sum += uint(histogram[i])
	}

	log2sum = float32(fastLog2(sum))
	missingSymbolSum = sum
	if !literalHistogram {
		for i = 0; i < histogramSize; i++ {
			if histogram[i] == 0 {
				missingSymbolSum++
			}
		}
	}

	missingSymbolCost = float32(fastLog2(missingSymbolSum)) + 2
	for i = 0; i < histogramSize; i++ {
		if histogram[i] == 0 {
			cost[i] = missingSymbolCost
			continue
		}

		/* Shannon bits for this symbol. */
		cost[i] = log2sum - float32(fastLog2(uint(histogram[i])))

		/* Cannot be coded with less than 1 bit */
		if cost[i] < 1 {
			cost[i] = 1
		}
	}
}

func zopfliCostModelSetFromCommands(self *zopfliCostModel, position uint, ringbuffer []byte, ringbufferMask uint, commands []command, lastInsertLen uint) {
	var histogramLiteralVar [numLiteralSymbols]uint32
	var histogramCmd [numCommandSymbols]uint32
	var histogramDist [maxEffectiveDistanceAlphabetSize]uint32
	var costLiteral [numLiteralSymbols]float32
	var pos uint = position - lastInsertLen
	var minCostCmd float32 = kInfinity
	var costCmd []float32 = self.costCmd[:]
	var literalCosts []float32

	histogramLiteralVar = [numLiteralSymbols]uint32{}
	histogramCmd = [numCommandSymbols]uint32{}
	histogramDist = [maxEffectiveDistanceAlphabetSize]uint32{}

	for i := range commands {
		var inslength uint = uint(commands[i].insert_len_)
		var copylength uint = uint(commandCopyLen(&commands[i]))
		var distcode uint = uint(commands[i].dist_prefix_) & 0x3FF
		var cmdcode uint = uint(commands[i].cmd_prefix_)
		var j uint

		histogramCmd[cmdcode]++
		if cmdcode >= 128 {
			histogramDist[distcode]++
		}

		for j = 0; j < inslength; j++ {
			histogramLiteralVar[ringbuffer[(pos+j)&ringbufferMask]]++
		}

		pos += inslength + copylength
	}

	setCost(histogramLiteralVar[:], numLiteralSymbols, true, costLiteral[:])
	setCost(histogramCmd[:], numCommandSymbols, false, costCmd)
	setCost(histogramDist[:], uint(self.distanceHistogramSize), false, self.costDist)

	for i := 0; i < numCommandSymbols; i++ {
		minCostCmd = brotliMinFloat(minCostCmd, costCmd[i])
	}

	self.minCostCmd = minCostCmd
	{
		literalCosts = self.literalCosts
		var literalCarry float32 = 0.0
		numBytes := int(self.numBytes)
		literalCosts[0] = 0.0
		for i := 0; i < numBytes; i++ {
			literalCarry += costLiteral[ringbuffer[(position+uint(i))&ringbufferMask]]
			literalCosts[i+1] = literalCosts[i] + literalCarry
			literalCarry -= literalCosts[i+1] - literalCosts[i]
		}
	}
}

func zopfliCostModelSetFromLiteralCosts(self *zopfliCostModel, position uint, ringbuffer []byte, ringbufferMask uint) {
	var literalCosts []float32 = self.literalCosts
	var literalCarry float32 = 0.0
	var costDist []float32 = self.costDist
	var costCmd []float32 = self.costCmd[:]
	var numBytes uint = self.numBytes
	var i uint
	estimateBitCostsForLiterals(position, numBytes, ringbufferMask, ringbuffer, literalCosts[1:])
	literalCosts[0] = 0.0
	for i = 0; i < numBytes; i++ {
		literalCarry += literalCosts[i+1]
		literalCosts[i+1] = literalCosts[i] + literalCarry
		literalCarry -= literalCosts[i+1] - literalCosts[i]
	}

	for i = 0; i < numCommandSymbols; i++ {
		costCmd[i] = float32(fastLog2(uint(11 + uint32(i))))
	}

	for i = 0; uint32(i) < self.distanceHistogramSize; i++ {
		costDist[i] = float32(fastLog2(uint(20 + uint32(i))))
	}

	self.minCostCmd = float32(fastLog2(11))
}

func zopfliCostModelGetCommandCost(self *zopfliCostModel, cmdcode uint16) float32 {
	return self.costCmd[cmdcode]
}

func zopfliCostModelGetDistanceCost(self *zopfliCostModel, distcode uint) float32 {
	return self.costDist[distcode]
}

func zopfliCostModelGetLiteralCosts(self *zopfliCostModel, from uint, to uint) float32 {
	return self.literalCosts[to] - self.literalCosts[from]
}

func zopfliCostModelGetMinCostCmd(self *zopfliCostModel) float32 {
	return self.minCostCmd
}

/* REQUIRES: len >= 2, start_pos <= pos */
/* REQUIRES: cost < kInfinity, nodes[start_pos].cost < kInfinity */
/* Maintains the "ZopfliNode array invariant". */
func updateZopfliNode(nodes []zopfliNode, pos uint, startPos uint, len uint, lenCode uint, dist uint, shortCode uint, cost float32) {
	var next *zopfliNode = &nodes[pos+len]
	next.length = uint32(len | (len+9-lenCode)<<25)
	next.distance = uint32(dist)
	next.dcodeInsertLength = uint32(shortCode<<27 | (pos - startPos))
	next.u.cost = cost
}

type posData struct {
	pos           uint
	distanceCache [4]int
	costdiff      float32
	cost          float32
}

/* Maintains the smallest 8-cost difference together with their positions */
type startPosQueue struct {
	q_   [8]posData
	idx_ uint
}

func initStartPosQueue(self *startPosQueue) {
	self.idx_ = 0
}

func startPosQueueSize(self *startPosQueue) uint {
	return brotliMinSizeT(self.idx_, 8)
}

func startPosQueuePush(self *startPosQueue, posdata *posData) {
	var offset uint = ^(self.idx_) & 7
	self.idx_++
	var queueSize uint = startPosQueueSize(self)
	var i uint
	var q []posData = self.q_[:]
	q[offset] = *posdata

	/* Restore the sorted order. In the list of |queueSize| items at most |queueSize - 1|
	   adjacent element comparisons / swaps are required. */
	for i = 1; i < queueSize; i++ {
		if q[offset&7].costdiff > q[(offset+1)&7].costdiff {
			var tmp posData = q[offset&7]
			q[offset&7] = q[(offset+1)&7]
			q[(offset+1)&7] = tmp
		}

		offset++
	}
}

func startPosQueueAt(self *startPosQueue, k uint) *posData {
	return &self.q_[(k-self.idx_)&7]
}

/* Returns the minimum possible copy length that can improve the cost of any */
/* future position. */
func computeMinimumCopyLength(startCost float32, nodes []zopfliNode, numBytes uint, pos uint) uint {
	var minCost float32 = startCost
	var l uint = 2
	var nextLenBucket uint = 4
	/* Compute the minimum possible cost of reaching any future position. */

	var nextLenOffset uint = 10
	for pos+l <= numBytes && nodes[pos+l].u.cost <= minCost {
		/* We already reached (pos + l) with no more cost than the minimum
		   possible cost of reaching anything from this pos, so there is no point in
		   looking for lengths <= l. */
		l++

		if l == nextLenOffset {
			/* We reached the next copy length code bucket, so we add one more
			   extra bit to the minimum cost. */
			minCost += 1.0

			nextLenOffset += nextLenBucket
			nextLenBucket *= 2
		}
	}

	return uint(l)
}

/*
REQUIRES: nodes[pos].cost < kInfinity

	REQUIRES: nodes[0..pos] satisfies that "ZopfliNode array invariant".
*/
func computeDistanceShortcut(blockStart uint, pos uint, maxBackwardLimit uint, gap uint, nodes []zopfliNode) uint32 {
	var clen uint = uint(zopfliNodeCopyLength(&nodes[pos]))
	var ilen uint = uint(nodes[pos].dcodeInsertLength & 0x7FFFFFF)
	var dist uint = uint(zopfliNodeCopyDistance(&nodes[pos]))

	/* Since |block_start + pos| is the end position of the command, the copy part
	   starts from |block_start + pos - clen|. Distances that are greater than
	   this or greater than |max_backward_limit| + |gap| are static dictionary
	   references, and do not update the last distances.
	   Also distance code 0 (last distance) does not update the last distances. */
	if pos == 0 {
		return 0
	} else if dist+clen <= blockStart+pos+gap && dist <= maxBackwardLimit+gap && zopfliNodeDistanceCode(&nodes[pos]) > 0 {
		return uint32(pos)
	} else {
		return nodes[pos-clen-ilen].u.shortcut
	}
}

/*
Fills in dist_cache[0..3] with the last four distances (as defined by

	Section 4. of the Spec) that would be used at (block_start + pos) if we
	used the shortest path of commands from block_start, computed from
	nodes[0..pos]. The last four distances at block_start are in
	starting_dist_cache[0..3].
	REQUIRES: nodes[pos].cost < kInfinity
	REQUIRES: nodes[0..pos] satisfies that "ZopfliNode array invariant".
*/
func computeDistanceCache(pos uint, startingDistCache []int, nodes []zopfliNode, distCache []int) {
	var idx int = 0
	var p uint = uint(nodes[pos].u.shortcut)
	for idx < 4 && p > 0 {
		var ilen uint = uint(nodes[p].dcodeInsertLength & 0x7FFFFFF)
		var clen uint = uint(zopfliNodeCopyLength(&nodes[p]))
		var dist uint = uint(zopfliNodeCopyDistance(&nodes[p]))
		distCache[idx] = int(dist)
		idx++

		/* Because of prerequisite, p >= clen + ilen >= 2. */
		p = uint(nodes[p-clen-ilen].u.shortcut)
	}

	for ; idx < 4; idx++ {
		distCache[idx] = startingDistCache[0]
		startingDistCache = startingDistCache[1:]
	}
}

/*
Maintains "ZopfliNode array invariant" and pushes node to the queue, if it

	is eligible.
*/
func evaluateNode(blockStart uint, pos uint, maxBackwardLimit uint, gap uint, startingDistCache []int, model *zopfliCostModel, queue *startPosQueue, nodes []zopfliNode) {
	/* Save cost, because ComputeDistanceCache invalidates it. */
	var nodeCost float32 = nodes[pos].u.cost
	nodes[pos].u.shortcut = computeDistanceShortcut(blockStart, pos, maxBackwardLimit, gap, nodes)
	if nodeCost <= zopfliCostModelGetLiteralCosts(model, 0, pos) {
		var posdata posData
		posdata.pos = pos
		posdata.cost = nodeCost
		posdata.costdiff = nodeCost - zopfliCostModelGetLiteralCosts(model, 0, pos)
		computeDistanceCache(pos, startingDistCache, nodes, posdata.distanceCache[:])
		startPosQueuePush(queue, &posdata)
	}
}

/* Returns longest copy length. */
func updateNodes(numBytes uint, blockStart uint, pos uint, ringbuffer []byte, ringbufferMask uint, params *encoderParams, maxBackwardLimit uint, startingDistCache []int, numMatches uint, matches []backwardMatch, model *zopfliCostModel, queue *startPosQueue, nodes []zopfliNode) uint {
	var curIx uint = blockStart + pos
	var curIxMasked uint = curIx & ringbufferMask
	var maxDistance uint = brotliMinSizeT(curIx, maxBackwardLimit)
	var maxLen uint = numBytes - pos
	var maxZopfliLenVar uint = maxZopfliLen(params)
	var maxIters uint = maxZopfliCandidates(params)
	var minLen uint
	var result uint = 0
	var k uint
	var gap uint = 0

	evaluateNode(blockStart, pos, maxBackwardLimit, gap, startingDistCache, model, queue, nodes)
	{
		var posdata *posData = startPosQueueAt(queue, 0)
		var minCost float32 = posdata.cost + zopfliCostModelGetMinCostCmd(model) + zopfliCostModelGetLiteralCosts(model, posdata.pos, pos)
		minLen = computeMinimumCopyLength(minCost, nodes, numBytes, pos)
	}

	/* Go over the command starting positions in order of increasing cost
	   difference. */
	for k = 0; k < maxIters && k < startPosQueueSize(queue); k++ {
		var posdata *posData = startPosQueueAt(queue, k)
		var start uint = posdata.pos
		var inscode uint16 = getInsertLengthCode(pos - start)
		var startCostdiff float32 = posdata.costdiff
		var baseCost float32 = startCostdiff + float32(getInsertExtra(inscode)) + zopfliCostModelGetLiteralCosts(model, 0, pos)
		var bestLen uint = minLen - 1
		var j uint = 0
		/* Look for last distance matches using the distance cache from this
		   starting position. */
		for ; j < numDistanceShortCodes && bestLen < maxLen; j++ {
			var idx uint = uint(kDistanceCacheIndex[j])
			var backward uint = uint(posdata.distanceCache[idx] + kDistanceCacheOffset[j])
			var prevIx uint = curIx - backward
			var lenV uint = 0
			var continuation byte = ringbuffer[curIxMasked+bestLen]
			if curIxMasked+bestLen > ringbufferMask {
				break
			}

			if backward > maxDistance+gap {
				/* Word dictionary -> ignore. */
				continue
			}

			if backward <= maxDistance {
				/* Regular backward reference. */
				if prevIx >= curIx {
					continue
				}

				prevIx &= ringbufferMask
				if prevIx+bestLen > ringbufferMask || continuation != ringbuffer[prevIx+bestLen] {
					continue
				}

				lenV = findMatchLengthWithLimit(ringbuffer[prevIx:], ringbuffer[curIxMasked:], maxLen)
			} else {
				continue
			}
			{
				var distCost float32 = baseCost + zopfliCostModelGetDistanceCost(model, j)
				var l uint
				for l = bestLen + 1; l <= lenV; l++ {
					var copycode uint16 = getCopyLengthCode(l)
					var cmdcode uint16 = combineLengthCodes(inscode, copycode, j == 0)
					var tmp float32
					if cmdcode < 128 {
						tmp = baseCost
					} else {
						tmp = distCost
					}
					var cost float32 = tmp + float32(getCopyExtra(copycode)) + zopfliCostModelGetCommandCost(model, cmdcode)
					if cost < nodes[pos+l].u.cost {
						updateZopfliNode(nodes, pos, start, l, l, backward, j+1, cost)
						result = brotliMaxSizeT(result, l)
					}

					bestLen = l
				}
			}
		}

		/* At higher iterations look only for new last distance matches, since
		   looking only for new command start positions with the same distances
		   does not help much. */
		if k >= 2 {
			continue
		}
		{
			/* Loop through all possible copy lengths at this position. */
			var lenV uint = minLen
			for j = 0; j < numMatches; j++ {
				var match backwardMatch = matches[j]
				var dist uint = uint(match.distance)
				var isDictionaryMatch bool = dist > maxDistance+gap
				var distCode uint = dist + numDistanceShortCodes - 1
				var distSymbol uint16
				var distextra uint32
				var distnumextra uint32
				var distCost float32
				var maxMatchLen uint
				/* We already tried all possible last distance matches, so we can use
				   normal distance code here. */
				prefixEncodeCopyDistance(distCode, uint(params.dist.numDirectDistanceCodes), uint(params.dist.distancePostfixBits), &distSymbol, &distextra)

				distnumextra = uint32(distSymbol) >> 10
				distCost = baseCost + float32(distnumextra) + zopfliCostModelGetDistanceCost(model, uint(distSymbol)&0x3FF)

				/* Try all copy lengths up until the maximum copy length corresponding
				   to this distance. If the distance refers to the static dictionary, or
				   the maximum length is long enough, try only one maximum length. */
				maxMatchLen = backwardMatchLength(&match)

				if lenV < maxMatchLen && (isDictionaryMatch || maxMatchLen > maxZopfliLenVar) {
					lenV = maxMatchLen
				}

				for ; lenV <= maxMatchLen; lenV++ {
					var lenCode uint
					if isDictionaryMatch {
						lenCode = backwardMatchLengthCode(&match)
					} else {
						lenCode = lenV
					}
					var copycode uint16 = getCopyLengthCode(lenCode)
					var cmdcode uint16 = combineLengthCodes(inscode, copycode, false)
					var cost float32 = distCost + float32(getCopyExtra(copycode)) + zopfliCostModelGetCommandCost(model, cmdcode)
					if cost < nodes[pos+lenV].u.cost {
						updateZopfliNode(nodes, pos, start, uint(lenV), lenCode, dist, 0, cost)
						if lenV > result {
							result = lenV
						}
					}
				}
			}
		}
	}

	return result
}

func computeShortestPathFromNodes(numBytes uint, nodes []zopfliNode) uint {
	var index uint = numBytes
	var numCommands uint = 0
	for nodes[index].dcodeInsertLength&0x7FFFFFF == 0 && nodes[index].length == 1 {
		index--
	}
	nodes[index].u.next = math.MaxUint32
	for index != 0 {
		var lenV uint = uint(zopfliNodeCommandLength(&nodes[index]))
		index -= uint(lenV)
		nodes[index].u.next = uint32(lenV)
		numCommands++
	}

	return numCommands
}

/* REQUIRES: nodes != NULL and len(nodes) >= num_bytes + 1 */
func zopfliCreateCommands(numBytes uint, blockStart uint, nodes []zopfliNode, distCache []int, lastInsertLen *uint, params *encoderParams, commands *[]command, numLiterals *uint) {
	var maxBackwardLimit uint = maxBackwardLimitFn(params.lgwin)
	var pos uint = 0
	var offset uint32 = nodes[0].u.next
	var i uint
	var gap uint = 0
	for i = 0; offset != math.MaxUint32; i++ {
		var next *zopfliNode = &nodes[uint32(pos)+offset]
		var copyLength uint = uint(zopfliNodeCopyLength(next))
		var insertLength uint = uint(next.dcodeInsertLength & 0x7FFFFFF)
		pos += insertLength
		offset = next.u.next
		if i == 0 {
			insertLength += *lastInsertLen
			*lastInsertLen = 0
		}
		{
			var distance uint = uint(zopfliNodeCopyDistance(next))
			var lenCode uint = uint(zopfliNodeLengthCode(next))
			var maxDistance uint = brotliMinSizeT(blockStart+pos, maxBackwardLimit)
			var isDictionary bool = distance > maxDistance+gap
			var distCode uint = uint(zopfliNodeDistanceCode(next))
			*commands = append(*commands, makeCommand(&params.dist, insertLength, copyLength, int(lenCode)-int(copyLength), distCode))

			if !isDictionary && distCode > 0 {
				distCache[3] = distCache[2]
				distCache[2] = distCache[1]
				distCache[1] = distCache[0]
				distCache[0] = int(distance)
			}
		}

		*numLiterals += insertLength
		pos += copyLength
	}

	*lastInsertLen += numBytes - pos
}

func zopfliIterate(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint, params *encoderParams, gap uint, distCache []int, model *zopfliCostModel, numMatches []uint32, matches []backwardMatch, nodes []zopfliNode) uint {
	var maxBackwardLimit uint = maxBackwardLimitFn(params.lgwin)
	var maxZopfliLenV uint = maxZopfliLen(params)
	var queue startPosQueue
	var curMatchPos uint = 0
	var i uint
	nodes[0].length = 0
	nodes[0].u.cost = 0
	initStartPosQueue(&queue)
	for i = 0; i+3 < numBytes; i++ {
		var skip uint = updateNodes(numBytes, position, i, ringbuffer, ringbufferMask, params, maxBackwardLimit, distCache, uint(numMatches[i]), matches[curMatchPos:], model, &queue, nodes)
		if skip < longCopyQuickStep {
			skip = 0
		}
		curMatchPos += uint(numMatches[i])
		if numMatches[i] == 1 && backwardMatchLength(&matches[curMatchPos-1]) > maxZopfliLenV {
			skip = brotliMaxSizeT(backwardMatchLength(&matches[curMatchPos-1]), skip)
		}

		if skip > 1 {
			skip--
			for skip != 0 {
				i++
				if i+3 >= numBytes {
					break
				}
				evaluateNode(position, i, maxBackwardLimit, gap, distCache, model, &queue, nodes)
				curMatchPos += uint(numMatches[i])
				skip--
			}
		}
	}

	return computeShortestPathFromNodes(numBytes, nodes)
}

/*
Computes the shortest path of commands from position to at most

	position + num_bytes.

	On return, path->size() is the number of commands found and path[i] is the
	length of the i-th command (copy length plus insert length).
	Note that the sum of the lengths of all commands can be less than num_bytes.

	On return, the nodes[0..num_bytes] array will have the following
	"ZopfliNode array invariant":
	For each i in [1..num_bytes], if nodes[i].cost < kInfinity, then
	  (1) nodes[i].copy_length() >= 2
	  (2) nodes[i].command_length() <= i and
	  (3) nodes[i - nodes[i].command_length()].cost < kInfinity

REQUIRES: nodes != nil and len(nodes) >= num_bytes + 1
*/
func zopfliComputeShortestPath(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint, params *encoderParams, distCache []int, hasher *h10, nodes []zopfliNode) uint {
	var maxBackwardLimit uint = maxBackwardLimitFn(params.lgwin)
	var maxZopfliLenV uint = maxZopfliLen(params)
	var model zopfliCostModel
	var queue startPosQueue
	var matches [2 * (maxNumMatchesH10 + 64)]backwardMatch
	var storeEnd uint
	if numBytes >= hasher.StoreLookahead() {
		storeEnd = position + numBytes - hasher.StoreLookahead() + 1
	} else {
		storeEnd = position
	}
	var i uint
	var gap uint = 0
	var lzMatchesOffset uint = 0
	nodes[0].length = 0
	nodes[0].u.cost = 0
	initZopfliCostModel(&model, &params.dist, numBytes)
	zopfliCostModelSetFromLiteralCosts(&model, position, ringbuffer, ringbufferMask)
	initStartPosQueue(&queue)
	for i = 0; i+hasher.HashTypeLength()-1 < numBytes; i++ {
		var pos uint = position + i
		var maxDistance uint = brotliMinSizeT(pos, maxBackwardLimit)
		var skip uint
		var numMatches uint
		numMatches = findAllMatchesH10(hasher, &params.dictionary, ringbuffer, ringbufferMask, pos, numBytes-i, maxDistance, gap, params, matches[lzMatchesOffset:])
		if numMatches > 0 && backwardMatchLength(&matches[numMatches-1]) > maxZopfliLenV {
			matches[0] = matches[numMatches-1]
			numMatches = 1
		}

		skip = updateNodes(numBytes, position, i, ringbuffer, ringbufferMask, params, maxBackwardLimit, distCache, numMatches, matches[:], &model, &queue, nodes)
		if skip < longCopyQuickStep {
			skip = 0
		}
		if numMatches == 1 && backwardMatchLength(&matches[0]) > maxZopfliLenV {
			skip = brotliMaxSizeT(backwardMatchLength(&matches[0]), skip)
		}

		if skip > 1 {
			/* Add the tail of the copy to the hasher. */
			hasher.StoreRange(ringbuffer, ringbufferMask, pos+1, brotliMinSizeT(pos+skip, storeEnd))

			skip--
			for skip != 0 {
				i++
				if i+hasher.HashTypeLength()-1 >= numBytes {
					break
				}
				evaluateNode(position, i, maxBackwardLimit, gap, distCache, &model, &queue, nodes)
				skip--
			}
		}
	}

	cleanupZopfliCostModel(&model)
	return computeShortestPathFromNodes(numBytes, nodes)
}

func createZopfliBackwardReferences(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint, params *encoderParams, hasher *h10, distCache []int, lastInsertLen *uint, commands *[]command, numLiterals *uint) {
	var nodes []zopfliNode
	nodes = make([]zopfliNode, numBytes+1)
	initZopfliNodes(nodes, numBytes+1)
	zopfliComputeShortestPath(numBytes, position, ringbuffer, ringbufferMask, params, distCache, hasher, nodes)
	zopfliCreateCommands(numBytes, position, nodes, distCache, lastInsertLen, params, commands, numLiterals)
	nodes = nil
}

func createHqZopfliBackwardReferences(numBytes uint, position uint, ringbuffer []byte, ringbufferMask uint, params *encoderParams, hasher hasherHandle, distCache []int, lastInsertLen *uint, commands *[]command, numLiterals *uint) {
	var maxBackwardLimit uint = maxBackwardLimitFn(params.lgwin)
	var numMatches []uint32 = make([]uint32, numBytes)
	var matchesSize uint = 4 * numBytes
	var storeEnd uint
	if numBytes >= hasher.StoreLookahead() {
		storeEnd = position + numBytes - hasher.StoreLookahead() + 1
	} else {
		storeEnd = position
	}
	var curMatchPos uint = 0
	var i uint
	var origNumLiterals uint
	var origLastInsertLen uint
	var origDistCache [4]int
	var origNumCommands int
	var model zopfliCostModel
	var nodes []zopfliNode
	var matches []backwardMatch = make([]backwardMatch, matchesSize)
	var gap uint = 0
	var shadowMatches uint = 0
	var newArray []backwardMatch
	for i = 0; i+hasher.HashTypeLength()-1 < numBytes; i++ {
		var pos uint = position + i
		var maxDistance uint = brotliMinSizeT(pos, maxBackwardLimit)
		var maxLength uint = numBytes - i
		var numFoundMatches uint
		var curMatchEnd uint
		var j uint

		/* Ensure that we have enough free slots. */
		if matchesSize < curMatchPos+maxNumMatchesH10+shadowMatches {
			var newSize uint = matchesSize
			if newSize == 0 {
				newSize = curMatchPos + maxNumMatchesH10 + shadowMatches
			}

			for newSize < curMatchPos+maxNumMatchesH10+shadowMatches {
				newSize *= 2
			}

			newArray = make([]backwardMatch, newSize)
			if matchesSize != 0 {
				copy(newArray, matches[:matchesSize])
			}

			matches = newArray
			matchesSize = newSize
		}

		numFoundMatches = findAllMatchesH10(hasher.(*h10), &params.dictionary, ringbuffer, ringbufferMask, pos, maxLength, maxDistance, gap, params, matches[curMatchPos+shadowMatches:])
		curMatchEnd = curMatchPos + numFoundMatches
		for j = curMatchPos; j+1 < curMatchEnd; j++ {
			assert(backwardMatchLength(&matches[j]) <= backwardMatchLength(&matches[j+1]))
		}

		numMatches[i] = uint32(numFoundMatches)
		if numFoundMatches > 0 {
			var matchLen uint = backwardMatchLength(&matches[curMatchEnd-1])
			if matchLen > maxZopfliLenQuality11 {
				var skip uint = matchLen - 1
				matches[curMatchPos] = matches[curMatchEnd-1]
				curMatchPos++
				numMatches[i] = 1

				/* Add the tail of the copy to the hasher. */
				hasher.StoreRange(ringbuffer, ringbufferMask, pos+1, brotliMinSizeT(pos+matchLen, storeEnd))
				var pos uint = i
				for i := 0; i < int(skip); i++ {
					numMatches[pos+1:][i] = 0
				}
				i += skip
			} else {
				curMatchPos = curMatchEnd
			}
		}
	}

	origNumLiterals = *numLiterals
	origLastInsertLen = *lastInsertLen
	copy(origDistCache[:], distCache[:4])
	origNumCommands = len(*commands)
	nodes = make([]zopfliNode, numBytes+1)
	initZopfliCostModel(&model, &params.dist, numBytes)
	for i = 0; i < 2; i++ {
		initZopfliNodes(nodes, numBytes+1)
		if i == 0 {
			zopfliCostModelSetFromLiteralCosts(&model, position, ringbuffer, ringbufferMask)
		} else {
			zopfliCostModelSetFromCommands(&model, position, ringbuffer, ringbufferMask, (*commands)[origNumCommands:], origLastInsertLen)
		}

		*commands = (*commands)[:origNumCommands]
		*numLiterals = origNumLiterals
		*lastInsertLen = origLastInsertLen
		copy(distCache, origDistCache[:4])
		zopfliIterate(numBytes, position, ringbuffer, ringbufferMask, params, gap, distCache, &model, numMatches, matches, nodes)
		zopfliCreateCommands(numBytes, position, nodes, distCache, lastInsertLen, params, commands, numLiterals)
	}

	cleanupZopfliCostModel(&model)
	nodes = nil
	matches = nil
	numMatches = nil
}
