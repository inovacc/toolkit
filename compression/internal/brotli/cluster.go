package brotli

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Functions for clustering similar histograms together. */

type histogramPair struct {
	idx1      uint32
	idx2      uint32
	costCombo float64
	costDiff  float64
}

func histogramPairIsLess(p1 *histogramPair, p2 *histogramPair) bool {
	if p1.costDiff != p2.costDiff {
		return p1.costDiff > p2.costDiff
	}

	return (p1.idx2 - p1.idx1) > (p2.idx2 - p2.idx1)
}

/* Returns entropy reduction of the context map when we combine two clusters. */
func clusterCostDiff(sizeA uint, sizeB uint) float64 {
	var sizeC uint = sizeA + sizeB
	return float64(sizeA)*fastLog2(sizeA) + float64(sizeB)*fastLog2(sizeB) - float64(sizeC)*fastLog2(sizeC)
}
