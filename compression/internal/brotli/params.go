package brotli

/* Copyright 2017 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Parameters for the Brotli encoder with chosen quality levels. */
type hasherParams struct {
	type_                   int
	bucketBits              int
	blockBits               int
	hashLen                 int
	numLastDistancesToCheck int
}

type distanceParams struct {
	distancePostfixBits    uint32
	numDirectDistanceCodes uint32
	alphabetSize           uint32
	maxDistance            uint
}

/* Encoding parameters */
type encoderParams struct {
	mode                          int
	quality                       int
	lgwin                         uint
	lgblock                       int
	sizeHint                      uint
	disableLiteralContextModeling bool
	largeWindow                   bool
	hasher                        hasherParams
	dist                          distanceParams
	dictionary                    encoderDictionary
}
