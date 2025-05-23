package brotli

import "math"

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

func initialEntropyCodesCommand(data []uint16, length uint, stride uint, num_histograms uint, histograms []histogramCommand) {
	var seed uint32 = 7
	var block_length uint = length / num_histograms
	var i uint
	clearHistogramsCommand(histograms, num_histograms)
	for i = 0; i < num_histograms; i++ {
		var pos uint = length * i / num_histograms
		if i != 0 {
			pos += uint(myRand(&seed) % uint32(block_length))
		}

		if pos+stride >= length {
			pos = length - stride - 1
		}

		histogramAddVectorCommand(&histograms[i], data[pos:], stride)
	}
}

func randomSampleCommand(seed *uint32, data []uint16, length uint, stride uint, sample *histogramCommand) {
	var pos uint = 0
	if stride >= length {
		stride = length
	} else {
		pos = uint(myRand(seed) % uint32(length-stride+1))
	}

	histogramAddVectorCommand(sample, data[pos:], stride)
}

func refineEntropyCodesCommand(data []uint16, length uint, stride uint, num_histograms uint, histograms []histogramCommand) {
	var iters uint = kIterMulForRefining*length/stride + kMinItersForRefining
	var seed uint32 = 7
	var iter uint
	iters = ((iters + num_histograms - 1) / num_histograms) * num_histograms
	for iter = 0; iter < iters; iter++ {
		var sample histogramCommand
		histogramClearCommand(&sample)
		randomSampleCommand(&seed, data, length, stride, &sample)
		histogramAddHistogramCommand(&histograms[iter%num_histograms], &sample)
	}
}

/*
Assigns a block id from the range [0, num_histograms) to each data element

	in data[0..length) and fills in block_id[0..length) with the assigned values.
	Returns the number of blocks, i.e. one plus the number of block switches.
*/
func findBlocksCommand(data []uint16, length uint, block_switch_bitcost float64, num_histograms uint, histograms []histogramCommand, insert_cost []float64, cost []float64, switch_signal []byte, block_id []byte) uint {
	var data_size uint = histogramDataSizeCommand()
	var bitmaplen uint = (num_histograms + 7) >> 3
	var num_blocks uint = 1
	var i uint
	var j uint
	assert(num_histograms <= 256)
	if num_histograms <= 1 {
		for i = 0; i < length; i++ {
			block_id[i] = 0
		}

		return 1
	}

	for i := 0; i < int(data_size*num_histograms); i++ {
		insert_cost[i] = 0
	}
	for i = 0; i < num_histograms; i++ {
		insert_cost[i] = fastLog2(uint(uint32(histograms[i].totalCount)))
	}

	for i = data_size; i != 0; {
		i--
		for j = 0; j < num_histograms; j++ {
			insert_cost[i*num_histograms+j] = insert_cost[j] - bitCost(uint(histograms[j].data_[i]))
		}
	}

	for i := 0; i < int(num_histograms); i++ {
		cost[i] = 0
	}
	for i := 0; i < int(length*bitmaplen); i++ {
		switch_signal[i] = 0
	}

	/* After each iteration of this loop, cost[k] will contain the difference
	   between the minimum cost of arriving at the current byte position using
	   entropy code k, and the minimum cost of arriving at the current byte
	   position. This difference is capped at the block switch cost, and if it
	   reaches block switch cost, it means that when we trace back from the last
	   position, we need to switch here. */
	for i = 0; i < length; i++ {
		var byte_ix uint = i
		var ix uint = byte_ix * bitmaplen
		var insert_cost_ix uint = uint(data[byte_ix]) * num_histograms
		var min_cost float64 = 1e99
		var block_switch_cost float64 = block_switch_bitcost
		var k uint
		for k = 0; k < num_histograms; k++ {
			/* We are coding the symbol in data[byte_ix] with entropy code k. */
			cost[k] += insert_cost[insert_cost_ix+k]

			if cost[k] < min_cost {
				min_cost = cost[k]
				block_id[byte_ix] = byte(k)
			}
		}

		/* More blocks for the beginning. */
		if byte_ix < 2000 {
			block_switch_cost *= 0.77 + 0.07*float64(byte_ix)/2000
		}

		for k = 0; k < num_histograms; k++ {
			cost[k] -= min_cost
			if cost[k] >= block_switch_cost {
				var mask byte = byte(1 << (k & 7))
				cost[k] = block_switch_cost
				assert(k>>3 < bitmaplen)
				switch_signal[ix+(k>>3)] |= mask
				/* Trace back from the last position and switch at the marked places. */
			}
		}
	}
	{
		var byte_ix uint = length - 1
		var ix uint = byte_ix * bitmaplen
		var cur_id byte = block_id[byte_ix]
		for byte_ix > 0 {
			var mask byte = byte(1 << (cur_id & 7))
			assert(uint(cur_id)>>3 < bitmaplen)
			byte_ix--
			ix -= bitmaplen
			if switch_signal[ix+uint(cur_id>>3)]&mask != 0 {
				if cur_id != block_id[byte_ix] {
					cur_id = block_id[byte_ix]
					num_blocks++
				}
			}

			block_id[byte_ix] = cur_id
		}
	}

	return num_blocks
}

var remapBlockIdsCommand_kInvalidId uint16 = 256

func remapBlockIdsCommand(block_ids []byte, length uint, new_id []uint16, num_histograms uint) uint {
	var next_id uint16 = 0
	var i uint
	for i = 0; i < num_histograms; i++ {
		new_id[i] = remapBlockIdsCommand_kInvalidId
	}

	for i = 0; i < length; i++ {
		assert(uint(block_ids[i]) < num_histograms)
		if new_id[block_ids[i]] == remapBlockIdsCommand_kInvalidId {
			new_id[block_ids[i]] = next_id
			next_id++
		}
	}

	for i = 0; i < length; i++ {
		block_ids[i] = byte(new_id[block_ids[i]])
		assert(uint(block_ids[i]) < num_histograms)
	}

	assert(uint(next_id) <= num_histograms)
	return uint(next_id)
}

func buildBlockHistogramsCommand(data []uint16, length uint, block_ids []byte, num_histograms uint, histograms []histogramCommand) {
	var i uint
	clearHistogramsCommand(histograms, num_histograms)
	for i = 0; i < length; i++ {
		histogramAddCommand(&histograms[block_ids[i]], uint(data[i]))
	}
}

var clusterBlocksCommand_kInvalidIndex uint32 = math.MaxUint32

func clusterBlocksCommand(data []uint16, length uint, num_blocks uint, block_ids []byte, split *blockSplit) {
	var histogram_symbols []uint32 = make([]uint32, num_blocks)
	var block_lengths []uint32 = make([]uint32, num_blocks)
	var expected_num_clusters uint = clustersPerBatch * (num_blocks + histogramsPerBatch - 1) / histogramsPerBatch
	var all_histograms_size uint = 0
	var all_histograms_capacity uint = expected_num_clusters
	var all_histograms []histogramCommand = make([]histogramCommand, all_histograms_capacity)
	var cluster_size_size uint = 0
	var cluster_size_capacity uint = expected_num_clusters
	var cluster_size []uint32 = make([]uint32, cluster_size_capacity)
	var num_clusters uint = 0
	var histograms []histogramCommand = make([]histogramCommand, brotliMinSizeT(num_blocks, histogramsPerBatch))
	var max_num_pairs uint = histogramsPerBatch * histogramsPerBatch / 2
	var pairs_capacity uint = max_num_pairs + 1
	var pairs []histogramPair = make([]histogramPair, pairs_capacity)
	var pos uint = 0
	var clusters []uint32
	var num_final_clusters uint
	var new_index []uint32
	var i uint
	var sizes = [histogramsPerBatch]uint32{0}
	var new_clusters = [histogramsPerBatch]uint32{0}
	var symbols = [histogramsPerBatch]uint32{0}
	var remap = [histogramsPerBatch]uint32{0}

	for i := 0; i < int(num_blocks); i++ {
		block_lengths[i] = 0
	}
	{
		var block_idx uint = 0
		for i = 0; i < length; i++ {
			assert(block_idx < num_blocks)
			block_lengths[block_idx]++
			if i+1 == length || block_ids[i] != block_ids[i+1] {
				block_idx++
			}
		}

		assert(block_idx == num_blocks)
	}

	for i = 0; i < num_blocks; i += histogramsPerBatch {
		var num_to_combine uint = brotliMinSizeT(num_blocks-i, histogramsPerBatch)
		var num_new_clusters uint
		var j uint
		for j = 0; j < num_to_combine; j++ {
			var k uint
			histogramClearCommand(&histograms[j])
			for k = 0; uint32(k) < block_lengths[i+j]; k++ {
				histogramAddCommand(&histograms[j], uint(data[pos]))
				pos++
			}

			histograms[j].bitCost = populationCostCommand(&histograms[j])
			new_clusters[j] = uint32(j)
			symbols[j] = uint32(j)
			sizes[j] = 1
		}

		num_new_clusters = histogramCombineCommand(histograms, sizes[:], symbols[:], new_clusters[:], []histogramPair(pairs), num_to_combine, num_to_combine, histogramsPerBatch, max_num_pairs)
		if all_histograms_capacity < (all_histograms_size + num_new_clusters) {
			var _new_size uint
			if all_histograms_capacity == 0 {
				_new_size = all_histograms_size + num_new_clusters
			} else {
				_new_size = all_histograms_capacity
			}
			var new_array []histogramCommand
			for _new_size < (all_histograms_size + num_new_clusters) {
				_new_size *= 2
			}
			new_array = make([]histogramCommand, _new_size)
			if all_histograms_capacity != 0 {
				copy(new_array, all_histograms[:all_histograms_capacity])
			}

			all_histograms = new_array
			all_histograms_capacity = _new_size
		}

		brotli_ensure_capacity_uint32_t(&cluster_size, &cluster_size_capacity, cluster_size_size+num_new_clusters)
		for j = 0; j < num_new_clusters; j++ {
			all_histograms[all_histograms_size] = histograms[new_clusters[j]]
			all_histograms_size++
			cluster_size[cluster_size_size] = sizes[new_clusters[j]]
			cluster_size_size++
			remap[new_clusters[j]] = uint32(j)
		}

		for j = 0; j < num_to_combine; j++ {
			histogram_symbols[i+j] = uint32(num_clusters) + remap[symbols[j]]
		}

		num_clusters += num_new_clusters
		assert(num_clusters == cluster_size_size)
		assert(num_clusters == all_histograms_size)
	}

	histograms = nil

	max_num_pairs = brotliMinSizeT(64*num_clusters, (num_clusters/2)*num_clusters)
	if pairs_capacity < max_num_pairs+1 {
		pairs = nil
		pairs = make([]histogramPair, (max_num_pairs + 1))
	}

	clusters = make([]uint32, num_clusters)
	for i = 0; i < num_clusters; i++ {
		clusters[i] = uint32(i)
	}

	num_final_clusters = histogramCombineCommand(all_histograms, cluster_size, histogram_symbols, clusters, pairs, num_clusters, num_blocks, maxNumberOfBlockTypes, max_num_pairs)
	pairs = nil
	cluster_size = nil

	new_index = make([]uint32, num_clusters)
	for i = 0; i < num_clusters; i++ {
		new_index[i] = clusterBlocksCommand_kInvalidIndex
	}
	pos = 0
	{
		var next_index uint32 = 0
		for i = 0; i < num_blocks; i++ {
			var histo histogramCommand
			var j uint
			var best_out uint32
			var best_bits float64
			histogramClearCommand(&histo)
			for j = 0; uint32(j) < block_lengths[i]; j++ {
				histogramAddCommand(&histo, uint(data[pos]))
				pos++
			}

			if i == 0 {
				best_out = histogram_symbols[0]
			} else {
				best_out = histogram_symbols[i-1]
			}
			best_bits = histogramBitCostDistanceCommand(&histo, &all_histograms[best_out])
			for j = 0; j < num_final_clusters; j++ {
				var cur_bits float64 = histogramBitCostDistanceCommand(&histo, &all_histograms[clusters[j]])
				if cur_bits < best_bits {
					best_bits = cur_bits
					best_out = clusters[j]
				}
			}

			histogram_symbols[i] = best_out
			if new_index[best_out] == clusterBlocksCommand_kInvalidIndex {
				new_index[best_out] = next_index
				next_index++
			}
		}
	}

	clusters = nil
	all_histograms = nil
	brotli_ensure_capacity_uint8_t(&split.types, &split.typesAllocSize, num_blocks)
	brotli_ensure_capacity_uint32_t(&split.lengths, &split.lengthsAllocSize, num_blocks)
	{
		var cur_length uint32 = 0
		var block_idx uint = 0
		var max_type byte = 0
		for i = 0; i < num_blocks; i++ {
			cur_length += block_lengths[i]
			if i+1 == num_blocks || histogram_symbols[i] != histogram_symbols[i+1] {
				var id byte = byte(new_index[histogram_symbols[i]])
				split.types[block_idx] = id
				split.lengths[block_idx] = cur_length
				max_type = brotliMaxUint8T(max_type, id)
				cur_length = 0
				block_idx++
			}
		}

		split.numBlocks = block_idx
		split.numTypes = uint(max_type) + 1
	}

	new_index = nil
	block_lengths = nil
	histogram_symbols = nil
}

func splitByteVectorCommand(data []uint16, literals_per_histogram uint, max_histograms uint, sampling_stride_length uint, block_switch_cost float64, params *encoderParams, split *blockSplit) {
	length := uint(len(data))
	var data_size uint = histogramDataSizeCommand()
	var num_histograms uint = length/literals_per_histogram + 1
	var histograms []histogramCommand
	if num_histograms > max_histograms {
		num_histograms = max_histograms
	}

	if length == 0 {
		split.numTypes = 1
		return
	} else if length < kMinLengthForBlockSplitting {
		brotli_ensure_capacity_uint8_t(&split.types, &split.typesAllocSize, split.numBlocks+1)
		brotli_ensure_capacity_uint32_t(&split.lengths, &split.lengthsAllocSize, split.numBlocks+1)
		split.numTypes = 1
		split.types[split.numBlocks] = 0
		split.lengths[split.numBlocks] = uint32(length)
		split.numBlocks++
		return
	}

	histograms = make([]histogramCommand, num_histograms)

	/* Find good entropy codes. */
	initialEntropyCodesCommand(data, length, sampling_stride_length, num_histograms, histograms)

	refineEntropyCodesCommand(data, length, sampling_stride_length, num_histograms, histograms)
	{
		var block_ids []byte = make([]byte, length)
		var num_blocks uint = 0
		var bitmaplen uint = (num_histograms + 7) >> 3
		var insert_cost []float64 = make([]float64, (data_size * num_histograms))
		var cost []float64 = make([]float64, num_histograms)
		var switch_signal []byte = make([]byte, (length * bitmaplen))
		var new_id []uint16 = make([]uint16, num_histograms)
		var iters uint
		if params.quality < hqZopflificationQuality {
			iters = 3
		} else {
			iters = 10
		}
		/* Find a good path through literals with the good entropy codes. */

		var i uint
		for i = 0; i < iters; i++ {
			num_blocks = findBlocksCommand(data, length, block_switch_cost, num_histograms, histograms, insert_cost, cost, switch_signal, block_ids)
			num_histograms = remapBlockIdsCommand(block_ids, length, new_id, num_histograms)
			buildBlockHistogramsCommand(data, length, block_ids, num_histograms, histograms)
		}

		insert_cost = nil
		cost = nil
		switch_signal = nil
		new_id = nil
		histograms = nil
		clusterBlocksCommand(data, length, num_blocks, block_ids, split)
		block_ids = nil
	}
}
