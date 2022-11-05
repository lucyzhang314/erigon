// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"errors"
	"fmt"
)

// List evm execution errors
var (
	// ErrInvalidSubroutineEntry means that a BEGINSUB was reached via iteration,
	// as opposed to from a JUMPSUB instruction
	ErrInvalidSubroutineEntry   = errors.New("invalid subroutine entry")
	ErrOutOfGas                 = errors.New("out of gas")
	ErrCodeStoreOutOfGas        = errors.New("contract creation code storage out of gas")
	ErrDepth                    = errors.New("max call depth exceeded")
	ErrInsufficientBalance      = errors.New("insufficient balance for transfer")
	ErrContractAddressCollision = errors.New("contract address collision")
	ErrExecutionReverted        = errors.New("execution reverted")
	ErrMaxInitCodeSizeExceeded  = errors.New("max initcode size exceeded")
	ErrMaxCodeSizeExceeded      = errors.New("max code size exceeded")
	ErrInvalidJump              = errors.New("invalid jump destination")
	ErrWriteProtection          = errors.New("write protection")
	ErrReturnDataOutOfBounds    = errors.New("return data out of bounds")
	ErrGasUintOverflow          = errors.New("gas uint64 overflow")
	ErrInvalidRetsub            = errors.New("invalid retsub")
	ErrReturnStackExceeded      = errors.New("return stack limit reached")
	ErrNonceUintOverflow        = errors.New("nonce uint64 overflow")
	ErrInvalidCode              = errors.New("invalid code: must not begin with 0xef")
	ErrInvalidEOFCode           = errors.New("invalid code: EOF validation failed")
	ErrInvalidInterpreter       = errors.New("invalid interpreter")

	// errStopToken is an internal token indicating interpreter loop termination,
	// never returned to outside callers.
	errStopToken = errors.New("stop token")
)

// EOF1 validation errors
var (
	ErrEOF1InvalidVersion                = errors.New("invalid version byte")
	ErrEOF1MultipleCodeSections          = errors.New("multiple code sections")
	ErrEOF1CodeSectionSizeMissing        = errors.New("can't read code section size")
	ErrEOF1EmptyCodeSection              = errors.New("code section size is 0")
	ErrEOF1DataSectionBeforeCodeSection  = errors.New("data section before code section")
	ErrEOF1MultipleDataSections          = errors.New("multiple data sections")
	ErrEOF1DataSectionSizeMissing        = errors.New("can't read data section size")
	ErrEOF1EmptyDataSection              = errors.New("data section size is 0")
	ErrEOF1UnknownSection                = errors.New("unknown section id")
	ErrEOF1CodeSectionMissing            = errors.New("no code section")
	ErrEOF1InvalidTotalSize              = errors.New("invalid total size")
	ErrEOF1UndefinedInstruction          = errors.New("undefined instruction")
	ErrEOF1TerminatingInstructionMissing = errors.New("code section doesn't end with terminating instruction")
	ErrEOF1InvalidRelativeOffset         = errors.New("relative offset points to immediate argument")
)

// ErrStackUnderflow wraps an evm error when the items on the stack less
// than the minimal requirement.
type ErrStackUnderflow struct {
	stackLen int
	required int
}

func (e *ErrStackUnderflow) Error() string {
	return fmt.Sprintf("stack underflow (%d <=> %d)", e.stackLen, e.required)
}

// ErrStackOverflow wraps an evm error when the items on the stack exceeds
// the maximum allowance.
type ErrStackOverflow struct {
	stackLen int
	limit    int
}

func (e *ErrStackOverflow) Error() string {
	return fmt.Sprintf("stack limit reached %d (%d)", e.stackLen, e.limit)
}

// ErrInvalidOpCode wraps an evm error when an invalid opcode is encountered.
type ErrInvalidOpCode struct {
	opcode OpCode
}

func (e *ErrInvalidOpCode) Error() string { return fmt.Sprintf("invalid opcode: %s", e.opcode) }
