package brotli

import "encoding/binary"

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Class to model the static dictionary. */

const maxStaticDictionaryMatchLen = 37

const kInvalidMatch uint32 = 0xFFFFFFF

/*
Copyright 2013 Google Inc. All Rights Reserved.

	Distributed under MIT license.
	See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/
func hash(data []byte) uint32 {
	var h uint32 = binary.LittleEndian.Uint32(data) * kDictHashMul32

	/* The higher bits contain more mixture from the multiplication,
	   so we take our results from there. */
	return h >> uint(32-kDictNumBits)
}

func addMatch(distance uint, len uint, lenCode uint, matches []uint32) {
	var match uint32 = uint32((distance << 5) + lenCode)
	matches[len] = brotliMinUint32T(matches[len], match)
}

func dictMatchLength(dict *dictionary, data []byte, id uint, len uint, maxlen uint) uint {
	var offset uint = uint(dict.offsets_by_length[len]) + len*id
	return findMatchLengthWithLimit(dict.data[offset:], data, brotliMinSizeT(uint(len), maxlen))
}

func isMatch(d *dictionary, w dictWord, data []byte, maxLength uint) bool {
	if uint(w.len) > maxLength {
		return false
	} else {
		var offset uint = uint(d.offsets_by_length[w.len]) + uint(w.len)*uint(w.idx)
		var dict []byte = d.data[offset:]
		if w.transform == 0 {
			/* Match against base dictionary word. */
			return findMatchLengthWithLimit(dict, data, uint(w.len)) == uint(w.len)
		} else if w.transform == 10 {
			/* Match against uppercase first transform.
			   Note that there are only ASCII uppercase words in the lookup table. */
			return dict[0] >= 'a' && dict[0] <= 'z' && (dict[0]^32) == data[0] && findMatchLengthWithLimit(dict[1:], data[1:], uint(w.len)-1) == uint(w.len-1)
		} else {
			/* Match against uppercase all transform.
			   Note that there are only ASCII uppercase words in the lookup table. */
			var i uint
			for i = 0; i < uint(w.len); i++ {
				if dict[i] >= 'a' && dict[i] <= 'z' {
					if (dict[i] ^ 32) != data[i] {
						return false
					}
				} else {
					if dict[i] != data[i] {
						return false
					}
				}
			}

			return true
		}
	}
}

func findAllStaticDictionaryMatches(dict *encoderDictionary, data []byte, minLength uint, maxLength uint, matches []uint32) bool {
	var hasFoundMatch bool = false
	{
		var offset uint = uint(dict.buckets[hash(data)])
		var end bool = offset == 0
		for !end {
			w := dict.dictWords[offset]
			offset++
			var l uint = uint(w.len) & 0x1F
			var n uint = uint(1) << dict.words.size_bits_by_length[l]
			var id uint = uint(w.idx)
			end = !(w.len&0x80 == 0)
			w.len = byte(l)
			if w.transform == 0 {
				var matchlen uint = dictMatchLength(dict.words, data, id, l, maxLength)
				var s []byte
				var minlen uint
				var maxlen uint
				var lenV uint

				/* Transform "" + BROTLI_TRANSFORM_IDENTITY + "" */
				if matchlen == l {
					addMatch(id, l, l, matches)
					hasFoundMatch = true
				}

				/* Transforms "" + BROTLI_TRANSFORM_OMIT_LAST_1 + "" and
				   "" + BROTLI_TRANSFORM_OMIT_LAST_1 + "ing " */
				if matchlen >= l-1 {
					addMatch(id+12*n, l-1, l, matches)
					if l+2 < maxLength && data[l-1] == 'i' && data[l] == 'n' && data[l+1] == 'g' && data[l+2] == ' ' {
						addMatch(id+49*n, l+3, l, matches)
					}

					hasFoundMatch = true
				}

				/* Transform "" + BROTLI_TRANSFORM_OMIT_LAST_# + "" (# = 2 .. 9) */
				minlen = minLength

				if l > 9 {
					minlen = brotliMaxSizeT(minlen, l-9)
				}
				maxlen = brotliMinSizeT(matchlen, l-2)
				for lenV = minlen; lenV <= maxlen; lenV++ {
					var cut uint = l - lenV
					var transformId uint = (cut << 2) + uint((dict.cutoffTransforms>>(cut*6))&0x3F)
					addMatch(id+transformId*n, uint(lenV), l, matches)
					hasFoundMatch = true
				}

				if matchlen < l || l+6 >= maxLength {
					continue
				}

				s = data[l:]

				/* Transforms "" + BROTLI_TRANSFORM_IDENTITY + <suffix> */
				if s[0] == ' ' {
					addMatch(id+n, l+1, l, matches)
					if s[1] == 'a' {
						if s[2] == ' ' {
							addMatch(id+28*n, l+3, l, matches)
						} else if s[2] == 's' {
							if s[3] == ' ' {
								addMatch(id+46*n, l+4, l, matches)
							}
						} else if s[2] == 't' {
							if s[3] == ' ' {
								addMatch(id+60*n, l+4, l, matches)
							}
						} else if s[2] == 'n' {
							if s[3] == 'd' && s[4] == ' ' {
								addMatch(id+10*n, l+5, l, matches)
							}
						}
					} else if s[1] == 'b' {
						if s[2] == 'y' && s[3] == ' ' {
							addMatch(id+38*n, l+4, l, matches)
						}
					} else if s[1] == 'i' {
						if s[2] == 'n' {
							if s[3] == ' ' {
								addMatch(id+16*n, l+4, l, matches)
							}
						} else if s[2] == 's' {
							if s[3] == ' ' {
								addMatch(id+47*n, l+4, l, matches)
							}
						}
					} else if s[1] == 'f' {
						if s[2] == 'o' {
							if s[3] == 'r' && s[4] == ' ' {
								addMatch(id+25*n, l+5, l, matches)
							}
						} else if s[2] == 'r' {
							if s[3] == 'o' && s[4] == 'm' && s[5] == ' ' {
								addMatch(id+37*n, l+6, l, matches)
							}
						}
					} else if s[1] == 'o' {
						if s[2] == 'f' {
							if s[3] == ' ' {
								addMatch(id+8*n, l+4, l, matches)
							}
						} else if s[2] == 'n' {
							if s[3] == ' ' {
								addMatch(id+45*n, l+4, l, matches)
							}
						}
					} else if s[1] == 'n' {
						if s[2] == 'o' && s[3] == 't' && s[4] == ' ' {
							addMatch(id+80*n, l+5, l, matches)
						}
					} else if s[1] == 't' {
						if s[2] == 'h' {
							if s[3] == 'e' {
								if s[4] == ' ' {
									addMatch(id+5*n, l+5, l, matches)
								}
							} else if s[3] == 'a' {
								if s[4] == 't' && s[5] == ' ' {
									addMatch(id+29*n, l+6, l, matches)
								}
							}
						} else if s[2] == 'o' {
							if s[3] == ' ' {
								addMatch(id+17*n, l+4, l, matches)
							}
						}
					} else if s[1] == 'w' {
						if s[2] == 'i' && s[3] == 't' && s[4] == 'h' && s[5] == ' ' {
							addMatch(id+35*n, l+6, l, matches)
						}
					}
				} else if s[0] == '"' {
					addMatch(id+19*n, l+1, l, matches)
					if s[1] == '>' {
						addMatch(id+21*n, l+2, l, matches)
					}
				} else if s[0] == '.' {
					addMatch(id+20*n, l+1, l, matches)
					if s[1] == ' ' {
						addMatch(id+31*n, l+2, l, matches)
						if s[2] == 'T' && s[3] == 'h' {
							if s[4] == 'e' {
								if s[5] == ' ' {
									addMatch(id+43*n, l+6, l, matches)
								}
							} else if s[4] == 'i' {
								if s[5] == 's' && s[6] == ' ' {
									addMatch(id+75*n, l+7, l, matches)
								}
							}
						}
					}
				} else if s[0] == ',' {
					addMatch(id+76*n, l+1, l, matches)
					if s[1] == ' ' {
						addMatch(id+14*n, l+2, l, matches)
					}
				} else if s[0] == '\n' {
					addMatch(id+22*n, l+1, l, matches)
					if s[1] == '\t' {
						addMatch(id+50*n, l+2, l, matches)
					}
				} else if s[0] == ']' {
					addMatch(id+24*n, l+1, l, matches)
				} else if s[0] == '\'' {
					addMatch(id+36*n, l+1, l, matches)
				} else if s[0] == ':' {
					addMatch(id+51*n, l+1, l, matches)
				} else if s[0] == '(' {
					addMatch(id+57*n, l+1, l, matches)
				} else if s[0] == '=' {
					if s[1] == '"' {
						addMatch(id+70*n, l+2, l, matches)
					} else if s[1] == '\'' {
						addMatch(id+86*n, l+2, l, matches)
					}
				} else if s[0] == 'a' {
					if s[1] == 'l' && s[2] == ' ' {
						addMatch(id+84*n, l+3, l, matches)
					}
				} else if s[0] == 'e' {
					if s[1] == 'd' {
						if s[2] == ' ' {
							addMatch(id+53*n, l+3, l, matches)
						}
					} else if s[1] == 'r' {
						if s[2] == ' ' {
							addMatch(id+82*n, l+3, l, matches)
						}
					} else if s[1] == 's' {
						if s[2] == 't' && s[3] == ' ' {
							addMatch(id+95*n, l+4, l, matches)
						}
					}
				} else if s[0] == 'f' {
					if s[1] == 'u' && s[2] == 'l' && s[3] == ' ' {
						addMatch(id+90*n, l+4, l, matches)
					}
				} else if s[0] == 'i' {
					if s[1] == 'v' {
						if s[2] == 'e' && s[3] == ' ' {
							addMatch(id+92*n, l+4, l, matches)
						}
					} else if s[1] == 'z' {
						if s[2] == 'e' && s[3] == ' ' {
							addMatch(id+100*n, l+4, l, matches)
						}
					}
				} else if s[0] == 'l' {
					if s[1] == 'e' {
						if s[2] == 's' && s[3] == 's' && s[4] == ' ' {
							addMatch(id+93*n, l+5, l, matches)
						}
					} else if s[1] == 'y' {
						if s[2] == ' ' {
							addMatch(id+61*n, l+3, l, matches)
						}
					}
				} else if s[0] == 'o' {
					if s[1] == 'u' && s[2] == 's' && s[3] == ' ' {
						addMatch(id+106*n, l+4, l, matches)
					}
				}
			} else {
				var isAllCaps bool = w.transform != transformUppercaseFirst
				/* Set is_all_caps=0 for BROTLI_TRANSFORM_UPPERCASE_FIRST and
				    is_all_caps=1 otherwise (BROTLI_TRANSFORM_UPPERCASE_ALL)
				transform. */

				var s []byte
				if !isMatch(dict.words, w, data, maxLength) {
					continue
				}

				/* Transform "" + kUppercase{First,All} + "" */
				var tmpVV int
				if isAllCaps {
					tmpVV = 44
				} else {
					tmpVV = 9
				}
				addMatch(id+uint(tmpVV)*n, l, l, matches)

				hasFoundMatch = true
				if l+1 >= maxLength {
					continue
				}

				/* Transforms "" + kUppercase{First,All} + <suffix> */
				s = data[l:]

				if s[0] == ' ' {
					var tmpVV int
					if isAllCaps {
						tmpVV = 68
					} else {
						tmpVV = 4
					}
					addMatch(id+uint(tmpVV)*n, l+1, l, matches)
				} else if s[0] == '"' {
					var tmpV int
					if isAllCaps {
						tmpV = 87
					} else {
						tmpV = 66
					}
					addMatch(id+uint(tmpV)*n, l+1, l, matches)
					if s[1] == '>' {
						var tmpVV int
						if isAllCaps {
							tmpVV = 97
						} else {
							tmpVV = 69
						}
						addMatch(id+uint(tmpVV)*n, l+2, l, matches)
					}
				} else if s[0] == '.' {
					var tmpV int
					if isAllCaps {
						tmpV = 101
					} else {
						tmpV = 79
					}
					addMatch(id+uint(tmpV)*n, l+1, l, matches)
					if s[1] == ' ' {
						var tmpV int
						if isAllCaps {
							tmpV = 114
						} else {
							tmpV = 88
						}
						addMatch(id+uint(tmpV)*n, l+2, l, matches)
					}
				} else if s[0] == ',' {
					var tmpV int
					if isAllCaps {
						tmpV = 112
					} else {
						tmpV = 99
					}
					addMatch(id+uint(tmpV)*n, l+1, l, matches)
					if s[1] == ' ' {
						var tmpV int
						if isAllCaps {
							tmpV = 107
						} else {
							tmpV = 58
						}
						addMatch(id+uint(tmpV)*n, l+2, l, matches)
					}
				} else if s[0] == '\'' {
					var tmpV int
					if isAllCaps {
						tmpV = 94
					} else {
						tmpV = 74
					}
					addMatch(id+uint(tmpV)*n, l+1, l, matches)
				} else if s[0] == '(' {
					var tmpV int
					if isAllCaps {
						tmpV = 113
					} else {
						tmpV = 78
					}
					addMatch(id+uint(tmpV)*n, l+1, l, matches)
				} else if s[0] == '=' {
					if s[1] == '"' {
						var tmpV int
						if isAllCaps {
							tmpV = 105
						} else {
							tmpV = 104
						}
						addMatch(id+uint(tmpV)*n, l+2, l, matches)
					} else if s[1] == '\'' {
						var tmpV int
						if isAllCaps {
							tmpV = 116
						} else {
							tmpV = 108
						}
						addMatch(id+uint(tmpV)*n, l+2, l, matches)
					}
				}
			}
		}
	}

	/* Transforms with prefixes " " and "." */
	if maxLength >= 5 && (data[0] == ' ' || data[0] == '.') {
		var isSpaceV bool = data[0] == ' '
		var offset uint = uint(dict.buckets[hash(data[1:])])
		var end bool = offset == 0
		for !end {
			w := dict.dictWords[offset]
			offset++
			var l uint = uint(w.len) & 0x1F
			var n uint = uint(1) << dict.words.size_bits_by_length[l]
			var id uint = uint(w.idx)
			end = !(w.len&0x80 == 0)
			w.len = byte(l)
			if w.transform == 0 {
				var s []byte
				if !isMatch(dict.words, w, data[1:], maxLength-1) {
					continue
				}

				/* Transforms " " + BROTLI_TRANSFORM_IDENTITY + "" and
				   "." + BROTLI_TRANSFORM_IDENTITY + "" */
				var tmpV int
				if isSpaceV {
					tmpV = 6
				} else {
					tmpV = 32
				}
				addMatch(id+uint(tmpV)*n, l+1, l, matches)

				hasFoundMatch = true
				if l+2 >= maxLength {
					continue
				}

				/* Transforms " " + BROTLI_TRANSFORM_IDENTITY + <suffix> and
				   "." + BROTLI_TRANSFORM_IDENTITY + <suffix>
				*/
				s = data[l+1:]

				if s[0] == ' ' {
					var tmpV int
					if isSpaceV {
						tmpV = 2
					} else {
						tmpV = 77
					}
					addMatch(id+uint(tmpV)*n, l+2, l, matches)
				} else if s[0] == '(' {
					var tmpV int
					if isSpaceV {
						tmpV = 89
					} else {
						tmpV = 67
					}
					addMatch(id+uint(tmpV)*n, l+2, l, matches)
				} else if isSpaceV {
					if s[0] == ',' {
						addMatch(id+103*n, l+2, l, matches)
						if s[1] == ' ' {
							addMatch(id+33*n, l+3, l, matches)
						}
					} else if s[0] == '.' {
						addMatch(id+71*n, l+2, l, matches)
						if s[1] == ' ' {
							addMatch(id+52*n, l+3, l, matches)
						}
					} else if s[0] == '=' {
						if s[1] == '"' {
							addMatch(id+81*n, l+3, l, matches)
						} else if s[1] == '\'' {
							addMatch(id+98*n, l+3, l, matches)
						}
					}
				}
			} else if isSpaceV {
				var isAllCaps bool = w.transform != transformUppercaseFirst
				/* Set is_all_caps=0 for BROTLI_TRANSFORM_UPPERCASE_FIRST and
				    is_all_caps=1 otherwise (BROTLI_TRANSFORM_UPPERCASE_ALL)
				transform. */

				var s []byte
				if !isMatch(dict.words, w, data[1:], maxLength-1) {
					continue
				}

				/* Transforms " " + kUppercase{First,All} + "" */
				var tmpV int
				if isAllCaps {
					tmpV = 85
				} else {
					tmpV = 30
				}
				addMatch(id+uint(tmpV)*n, l+1, l, matches)

				hasFoundMatch = true
				if l+2 >= maxLength {
					continue
				}

				/* Transforms " " + kUppercase{First,All} + <suffix> */
				s = data[l+1:]

				if s[0] == ' ' {
					var tmpV int
					if isAllCaps {
						tmpV = 83
					} else {
						tmpV = 15
					}
					addMatch(id+uint(tmpV)*n, l+2, l, matches)
				} else if s[0] == ',' {
					if !isAllCaps {
						addMatch(id+109*n, l+2, l, matches)
					}

					if s[1] == ' ' {
						var tmpV int
						if isAllCaps {
							tmpV = 111
						} else {
							tmpV = 65
						}
						addMatch(id+uint(tmpV)*n, l+3, l, matches)
					}
				} else if s[0] == '.' {
					var tmpV int
					if isAllCaps {
						tmpV = 115
					} else {
						tmpV = 96
					}
					addMatch(id+uint(tmpV)*n, l+2, l, matches)
					if s[1] == ' ' {
						var tmpV int
						if isAllCaps {
							tmpV = 117
						} else {
							tmpV = 91
						}
						addMatch(id+uint(tmpV)*n, l+3, l, matches)
					}
				} else if s[0] == '=' {
					if s[1] == '"' {
						var tmpV int
						if isAllCaps {
							tmpV = 110
						} else {
							tmpV = 118
						}
						addMatch(id+uint(tmpV)*n, l+3, l, matches)
					} else if s[1] == '\'' {
						var tmpV int
						if isAllCaps {
							tmpV = 119
						} else {
							tmpV = 120
						}
						addMatch(id+uint(tmpV)*n, l+3, l, matches)
					}
				}
			}
		}
	}

	if maxLength >= 6 {
		/* Transforms with prefixes "e ", "s ", ", " and "\xC2\xA0" */
		if (data[1] == ' ' && (data[0] == 'e' || data[0] == 's' || data[0] == ',')) || (data[0] == 0xC2 && data[1] == 0xA0) {
			var offset uint = uint(dict.buckets[hash(data[2:])])
			var end bool = offset == 0
			for !end {
				w := dict.dictWords[offset]
				offset++
				var l uint = uint(w.len) & 0x1F
				var n uint = uint(1) << dict.words.size_bits_by_length[l]
				var id uint = uint(w.idx)
				end = !(w.len&0x80 == 0)
				w.len = byte(l)
				if w.transform == 0 && isMatch(dict.words, w, data[2:], maxLength-2) {
					if data[0] == 0xC2 {
						addMatch(id+102*n, l+2, l, matches)
						hasFoundMatch = true
					} else if l+2 < maxLength && data[l+2] == ' ' {
						var t uint = 13
						if data[0] == 'e' {
							t = 18
						} else if data[0] == 's' {
							t = 7
						}
						addMatch(id+t*n, l+3, l, matches)
						hasFoundMatch = true
					}
				}
			}
		}
	}

	if maxLength >= 9 {
		/* Transforms with prefixes " the " and ".com/" */
		if (data[0] == ' ' && data[1] == 't' && data[2] == 'h' && data[3] == 'e' && data[4] == ' ') || (data[0] == '.' && data[1] == 'c' && data[2] == 'o' && data[3] == 'm' && data[4] == '/') {
			var offset uint = uint(dict.buckets[hash(data[5:])])
			var end bool = offset == 0
			for !end {
				w := dict.dictWords[offset]
				offset++
				var l uint = uint(w.len) & 0x1F
				var n uint = uint(1) << dict.words.size_bits_by_length[l]
				var id uint = uint(w.idx)
				end = !(w.len&0x80 == 0)
				w.len = byte(l)
				if w.transform == 0 && isMatch(dict.words, w, data[5:], maxLength-5) {
					var tmpV int
					if data[0] == ' ' {
						tmpV = 41
					} else {
						tmpV = 72
					}
					addMatch(id+uint(tmpV)*n, l+5, l, matches)
					hasFoundMatch = true
					if l+5 < maxLength {
						var s []byte = data[l+5:]
						if data[0] == ' ' {
							if l+8 < maxLength && s[0] == ' ' && s[1] == 'o' && s[2] == 'f' && s[3] == ' ' {
								addMatch(id+62*n, l+9, l, matches)
								if l+12 < maxLength && s[4] == 't' && s[5] == 'h' && s[6] == 'e' && s[7] == ' ' {
									addMatch(id+73*n, l+13, l, matches)
								}
							}
						}
					}
				}
			}
		}
	}

	return hasFoundMatch
}
