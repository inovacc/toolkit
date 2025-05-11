package brotli

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Functions for encoding of integers into prefix codes the amount of extra
   bits, and the actual values of the extra bits. */

/*
Here distance_code is an intermediate code, i.e. one of the special codes or

	the actual distance increased by BROTLI_NUM_DISTANCE_SHORT_CODES - 1.
*/
func prefixEncodeCopyDistance(distanceCode uint, numDirectCodes uint, postfixBits uint, code *uint16, extraBits *uint32) {
	if distanceCode < numDistanceShortCodes+numDirectCodes {
		*code = uint16(distanceCode)
		*extraBits = 0
		return
	} else {
		var dist uint = (uint(1) << (postfixBits + 2)) + (distanceCode - numDistanceShortCodes - numDirectCodes)
		var bucket uint = uint(log2FloorNonZero(dist) - 1)
		var postfixMask uint = (1 << postfixBits) - 1
		var postfix uint = dist & postfixMask
		var prefix uint = (dist >> bucket) & 1
		var offset uint = (2 + prefix) << bucket
		var nbits uint = bucket - postfixBits
		*code = uint16(nbits<<10 | (numDistanceShortCodes + numDirectCodes + ((2*(nbits-1) + prefix) << postfixBits) + postfix))
		*extraBits = uint32((dist - offset) >> postfixBits)
	}
}
