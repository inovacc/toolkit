package brotli

import "math"

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/*
Computes the bit cost reduction by combining out[idx1] and out[idx2] and if

	it is below a threshold, stores the pair (idx1, idx2) in the *pairs queue.
*/
func compareAndPushToQueueLiteral(out []histogramLiteral, cluster_size []uint32, idx1 uint32, idx2 uint32, max_num_pairs uint, pairs []histogramPair, num_pairs *uint) {
	var is_good_pair bool = false
	var p histogramPair
	p.idx2 = 0
	p.idx1 = p.idx2
	p.costCombo = 0
	p.costDiff = p.costCombo
	if idx1 == idx2 {
		return
	}

	if idx2 < idx1 {
		var t uint32 = idx2
		idx2 = idx1
		idx1 = t
	}

	p.idx1 = idx1
	p.idx2 = idx2
	p.costDiff = 0.5 * clusterCostDiff(uint(cluster_size[idx1]), uint(cluster_size[idx2]))
	p.costDiff -= out[idx1].bitCost
	p.costDiff -= out[idx2].bitCost

	if out[idx1].totalCount == 0 {
		p.costCombo = out[idx2].bitCost
		is_good_pair = true
	} else if out[idx2].totalCount == 0 {
		p.costCombo = out[idx1].bitCost
		is_good_pair = true
	} else {
		var threshold float64
		if *num_pairs == 0 {
			threshold = 1e99
		} else {
			threshold = brotliMaxDouble(0.0, pairs[0].costDiff)
		}
		var combo histogramLiteral = out[idx1]
		var cost_combo float64
		histogramAddHistogramLiteral(&combo, &out[idx2])
		cost_combo = populationCostLiteral(&combo)
		if cost_combo < threshold-p.costDiff {
			p.costCombo = cost_combo
			is_good_pair = true
		}
	}

	if is_good_pair {
		p.costDiff += p.costCombo
		if *num_pairs > 0 && histogramPairIsLess(&pairs[0], &p) {
			/* Replace the top of the queue if needed. */
			if *num_pairs < max_num_pairs {
				pairs[*num_pairs] = pairs[0]
				(*num_pairs)++
			}

			pairs[0] = p
		} else if *num_pairs < max_num_pairs {
			pairs[*num_pairs] = p
			(*num_pairs)++
		}
	}
}

func histogramCombineLiteral(out []histogramLiteral, cluster_size []uint32, symbols []uint32, clusters []uint32, pairs []histogramPair, num_clusters uint, symbols_size uint, max_clusters uint, max_num_pairs uint) uint {
	var cost_diff_threshold float64 = 0.0
	var min_cluster_size uint = 1
	var num_pairs uint = 0
	{
		/* We maintain a vector of histogram pairs, with the property that the pair
		   with the maximum bit cost reduction is the first. */
		var idx1 uint
		for idx1 = 0; idx1 < num_clusters; idx1++ {
			var idx2 uint
			for idx2 = idx1 + 1; idx2 < num_clusters; idx2++ {
				compareAndPushToQueueLiteral(out, cluster_size, clusters[idx1], clusters[idx2], max_num_pairs, pairs[0:], &num_pairs)
			}
		}
	}

	for num_clusters > min_cluster_size {
		var best_idx1 uint32
		var best_idx2 uint32
		var i uint
		if pairs[0].costDiff >= cost_diff_threshold {
			cost_diff_threshold = 1e99
			min_cluster_size = max_clusters
			continue
		}

		/* Take the best pair from the top of heap. */
		best_idx1 = pairs[0].idx1

		best_idx2 = pairs[0].idx2
		histogramAddHistogramLiteral(&out[best_idx1], &out[best_idx2])
		out[best_idx1].bitCost = pairs[0].costCombo
		cluster_size[best_idx1] += cluster_size[best_idx2]
		for i = 0; i < symbols_size; i++ {
			if symbols[i] == best_idx2 {
				symbols[i] = best_idx1
			}
		}

		for i = 0; i < num_clusters; i++ {
			if clusters[i] == best_idx2 {
				copy(clusters[i:], clusters[i+1:][:num_clusters-i-1])
				break
			}
		}

		num_clusters--
		{
			/* Remove pairs intersecting the just combined best pair. */
			var copy_to_idx uint = 0
			for i = 0; i < num_pairs; i++ {
				var p *histogramPair = &pairs[i]
				if p.idx1 == best_idx1 || p.idx2 == best_idx1 || p.idx1 == best_idx2 || p.idx2 == best_idx2 {
					/* Remove invalid pair from the queue. */
					continue
				}

				if histogramPairIsLess(&pairs[0], p) {
					/* Replace the top of the queue if needed. */
					var front histogramPair = pairs[0]
					pairs[0] = *p
					pairs[copy_to_idx] = front
				} else {
					pairs[copy_to_idx] = *p
				}

				copy_to_idx++
			}

			num_pairs = copy_to_idx
		}

		/* Push new pairs formed with the combined histogram to the heap. */
		for i = 0; i < num_clusters; i++ {
			compareAndPushToQueueLiteral(out, cluster_size, best_idx1, clusters[i], max_num_pairs, pairs[0:], &num_pairs)
		}
	}

	return num_clusters
}

/* What is the bit cost of moving histogram from cur_symbol to candidate. */
func histogramBitCostDistanceLiteral(histogram *histogramLiteral, candidate *histogramLiteral) float64 {
	if histogram.totalCount == 0 {
		return 0.0
	} else {
		var tmp histogramLiteral = *histogram
		histogramAddHistogramLiteral(&tmp, candidate)
		return populationCostLiteral(&tmp) - candidate.bitCost
	}
}

/*
Find the best 'out' histogram for each of the 'in' histograms.

	When called, clusters[0..num_clusters) contains the unique values from
	symbols[0..in_size), but this property is not preserved in this function.
	Note: we assume that out[]->bit_cost_ is already up-to-date.
*/
func histogramRemapLiteral(in []histogramLiteral, in_size uint, clusters []uint32, num_clusters uint, out []histogramLiteral, symbols []uint32) {
	var i uint
	for i = 0; i < in_size; i++ {
		var best_out uint32
		if i == 0 {
			best_out = symbols[0]
		} else {
			best_out = symbols[i-1]
		}
		var best_bits float64 = histogramBitCostDistanceLiteral(&in[i], &out[best_out])
		var j uint
		for j = 0; j < num_clusters; j++ {
			var cur_bits float64 = histogramBitCostDistanceLiteral(&in[i], &out[clusters[j]])
			if cur_bits < best_bits {
				best_bits = cur_bits
				best_out = clusters[j]
			}
		}

		symbols[i] = best_out
	}

	/* Recompute each out based on raw and symbols. */
	for i = 0; i < num_clusters; i++ {
		histogramClearLiteral(&out[clusters[i]])
	}

	for i = 0; i < in_size; i++ {
		histogramAddHistogramLiteral(&out[symbols[i]], &in[i])
	}
}

/* Reorders elements of the out[0..length) array and changes values in
   symbols[0..length) array in the following way:
     * when called, symbols[] contains indexes into out[], and has N unique
       values (possibly N < length)
     * on return, symbols'[i] = f(symbols[i]) and
                  out'[symbols'[i]] = out[symbols[i]], for each 0 <= i < length,
       where f is a bijection between the range of symbols[] and [0..N), and
       the first occurrences of values in symbols'[i] come in consecutive
       increasing order.
   Returns N, the number of unique values in symbols[]. */

var histogramReindexLiteral_kInvalidIndex uint32 = math.MaxUint32

func histogramReindexLiteral(out []histogramLiteral, symbols []uint32, length uint) uint {
	var new_index []uint32 = make([]uint32, length)
	var next_index uint32
	var tmp []histogramLiteral
	var i uint
	for i = 0; i < length; i++ {
		new_index[i] = histogramReindexLiteral_kInvalidIndex
	}

	next_index = 0
	for i = 0; i < length; i++ {
		if new_index[symbols[i]] == histogramReindexLiteral_kInvalidIndex {
			new_index[symbols[i]] = next_index
			next_index++
		}
	}

	/* TODO: by using idea of "cycle-sort" we can avoid allocation of
	   tmp and reduce the number of copying by the factor of 2. */
	tmp = make([]histogramLiteral, next_index)

	next_index = 0
	for i = 0; i < length; i++ {
		if new_index[symbols[i]] == next_index {
			tmp[next_index] = out[symbols[i]]
			next_index++
		}

		symbols[i] = new_index[symbols[i]]
	}

	new_index = nil
	for i = 0; uint32(i) < next_index; i++ {
		out[i] = tmp[i]
	}

	tmp = nil
	return uint(next_index)
}

func clusterHistogramsLiteral(in []histogramLiteral, in_size uint, max_histograms uint, out []histogramLiteral, out_size *uint, histogram_symbols []uint32) {
	var cluster_size []uint32 = make([]uint32, in_size)
	var clusters []uint32 = make([]uint32, in_size)
	var num_clusters uint = 0
	var max_input_histograms uint = 64
	var pairs_capacity uint = max_input_histograms * max_input_histograms / 2
	var pairs []histogramPair = make([]histogramPair, (pairs_capacity + 1))
	var i uint

	/* For the first pass of clustering, we allow all pairs. */
	for i = 0; i < in_size; i++ {
		cluster_size[i] = 1
	}

	for i = 0; i < in_size; i++ {
		out[i] = in[i]
		out[i].bitCost = populationCostLiteral(&in[i])
		histogram_symbols[i] = uint32(i)
	}

	for i = 0; i < in_size; i += max_input_histograms {
		var num_to_combine uint = brotliMinSizeT(in_size-i, max_input_histograms)
		var num_new_clusters uint
		var j uint
		for j = 0; j < num_to_combine; j++ {
			clusters[num_clusters+j] = uint32(i + j)
		}

		num_new_clusters = histogramCombineLiteral(out, cluster_size, histogram_symbols[i:], clusters[num_clusters:], pairs, num_to_combine, num_to_combine, max_histograms, pairs_capacity)
		num_clusters += num_new_clusters
	}
	{
		/* For the second pass, we limit the total number of histogram pairs.
		   After this limit is reached, we only keep searching for the best pair. */
		var max_num_pairs uint = brotliMinSizeT(64*num_clusters, (num_clusters/2)*num_clusters)
		if pairs_capacity < (max_num_pairs + 1) {
			var _new_size uint
			if pairs_capacity == 0 {
				_new_size = max_num_pairs + 1
			} else {
				_new_size = pairs_capacity
			}
			var new_array []histogramPair
			for _new_size < (max_num_pairs + 1) {
				_new_size *= 2
			}
			new_array = make([]histogramPair, _new_size)
			if pairs_capacity != 0 {
				copy(new_array, pairs[:pairs_capacity])
			}

			pairs = new_array
			pairs_capacity = _new_size
		}

		/* Collapse similar histograms. */
		num_clusters = histogramCombineLiteral(out, cluster_size, histogram_symbols, clusters, pairs, num_clusters, in_size, max_histograms, max_num_pairs)
	}

	pairs = nil
	cluster_size = nil

	/* Find the optimal map from original histograms to the final ones. */
	histogramRemapLiteral(in, in_size, clusters, num_clusters, out, histogram_symbols)

	clusters = nil

	/* Convert the context map to a canonical form. */
	*out_size = histogramReindexLiteral(out, histogram_symbols, in_size)
}
