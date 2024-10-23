// Copyright 2014 The go-ethereum Authors
// (original work)
// Copyright 2024 The Erigon Authors
// (modifications)
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package vm

const (
	set2BitsMask = uint64(0b11)
	set3BitsMask = uint64(0b111)
	set4BitsMask = uint64(0b1111)
	set5BitsMask = uint64(0b1_1111)
	set6BitsMask = uint64(0b11_1111)
	set7BitsMask = uint64(0b111_1111)
)

// TODO(racytech): fix this part. Make sure merge did not brake anything. If it did investigate.

// bitvec is a bit vector which maps bytes in a program.
// An unset bit means the byte is an opcode, a set bit means
// it's data (i.e. argument of PUSHxx).
// type bitvec []byte

// func (bits bitvec) set1(pos uint64) {
// 	bits[pos/8] |= 1 << (pos % 8)
// }

// func (bits bitvec) setN(flag uint16, pos uint64) {
// 	a := flag << (pos % 8)
// 	bits[pos/8] |= byte(a)
// 	if b := byte(a >> 8); b != 0 {
// 		bits[pos/8+1] = b
// 	}
// }

func (bits bitvec) set8(pos uint64) {
	a := uint64(0xFF << (pos % 8))
	bits[pos/8] |= a
	bits[pos/8+1] = ^a
}

// func (bits bitvec) set16(pos uint64) {
// 	a := byte(0xFF << (pos % 8))
// 	bits[pos/8] |= a
// 	bits[pos/8+1] = 0xFF
// 	bits[pos/8+2] = ^a
// }

// // codeSegment checks if the position is in a code segment.
// func (bits *bitvec) codeSegment(pos uint64) bool {
// 	return (((*bits)[pos/8] >> (pos % 8)) & 1) == 0
// }

// codeBitmap collects data locations in code.
func codeBitmap(code []byte) bitvec {
	// The bitmap is 4 bytes longer than necessary, in case the code
	// ends with a PUSH32, the algorithm will push zeroes onto the
	// bitvector outside the bounds of the actual code.
	bits := make(bitvec, (len(code)+32+63)/64)
	for pc := uint64(0); pc < uint64(len(code)); {
		op := OpCode(code[pc])
		pc++
		if int8(op) < int8(PUSH1) { // If not PUSH (the int8(op) > int(PUSH32) is always false).
			continue
		}
		if op == PUSH1 {
			bits.set1(pc)
			pc += 1
			continue
		}

		numbits := uint64(op - PUSH1 + 1)
		bits.setN(uint64(1)<<numbits-1, pc)
		pc += numbits
	}
	return bits
}

// bitvec is a bit vector which maps bytes in a program.
// An unset bit means the byte is an opcode, a set bit means
// it's data (i.e. argument of PUSHxx).
type bitvec []uint64

func (bits bitvec) set1(pos uint64) {
	bits[pos/64] |= 1 << (pos % 64)
}

func (bits bitvec) setN(flag uint64, pc uint64) {
	shift := pc % 64
	bits[pc/64] |= flag << shift
	if shift > 32 {
		bits[pc/64+1] = flag >> (64 - shift)
	}
}

// codeSegment checks if the position is in a code segment.
func (bits bitvec) codeSegment(pos uint64) bool {
	return ((bits[pos/64] >> (pos % 64)) & 1) == 0
}

// eofCodeBitmap collects data locations in code.
func eofCodeBitmap(code []byte) bitvec {
	// The bitmap is 4 bytes longer than necessary, in case the code
	// ends with a PUSH32, the algorithm will push zeroes onto the
	// bitvector outside the bounds of the actual code.
	bits := make(bitvec, len(code)/8+1+4)
	return eofCodeBitmapInternal(code, bits)
}

// eofCodeBitmapInternal is the internal implementation of codeBitmap for EOF
// code validation.
func eofCodeBitmapInternal(code []byte, bits bitvec) bitvec {
	for pc := uint64(0); pc < uint64(len(code)); {
		var (
			op      = OpCode(code[pc])
			numbits uint64
		)
		pc++

		switch {
		case op >= PUSH1 && op <= PUSH32:
			numbits = uint64(op - PUSH1 + 1)
		case op == RJUMP || op == RJUMPI:
			numbits = 2
		case op == RJUMPV:
			// RJUMPV is unique as it has a variable sized operand.
			// The total size is determined by the count byte which
			// immediate proceeds RJUMPV. Truncation will be caught
			// in other validation steps -- for now, just return a
			// valid bitmap for as much of the code as is
			// available.
			end := uint64(len(code))
			if pc >= end {
				// Count missing, no more bits to mark.
				return bits
			}
			numbits = uint64(code[pc]*2 + 1)
			if pc+uint64(numbits) > end {
				// Jump table is truncated, mark as many bits
				// as possible.
				numbits = uint64(end - pc)
			}
		default:
			// Op had no immediate operand, continue.
			continue
		}

		if numbits >= 8 { // TODO(racytech): fix this
			// for ; numbits >= 16; numbits -= 16 {
			// 	bits.setN(pc)
			// 	pc += 16
			// }
			// for ; numbits >= 8; numbits -= 8 {
			// 	bits.set8(pc)
			// 	pc += 8
			// }
		}
		switch numbits {
		case 1:
			bits.set1(pc)
			pc += 1
		case 2:
			bits.setN(set2BitsMask, pc)
			pc += 2
		case 3:
			bits.setN(set3BitsMask, pc)
			pc += 3
		case 4:
			bits.setN(set4BitsMask, pc)
			pc += 4
		case 5:
			bits.setN(set5BitsMask, pc)
			pc += 5
		case 6:
			bits.setN(set6BitsMask, pc)
			pc += 6
		case 7:
			bits.setN(set7BitsMask, pc)
			pc += 7
		}
	}
	return bits
}
