// Copyright (c) 2013 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcscript_test

import (
	"bytes"
	"fmt"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
	"github.com/conformal/seelog"
	"github.com/davecgh/go-spew/spew"
	"os"
	"testing"
)

// test scripts to test as many opcodes as possible.
// All run on a fake tx with a single in, single out.
type opcodeTest struct {
	script     []byte
	shouldPass bool
	shouldFail error
}

var opcodeTests = []opcodeTest{
	// does nothing, but doesn't put a true on the stack, should fail
	{script: []byte{btcscript.OP_NOP}, shouldPass: false},
	// should just put true on the stack, thus passes.
	{script: []byte{btcscript.OP_TRUE}, shouldPass: true},
	// should just put false on the stack, thus fails.
	{script: []byte{btcscript.OP_FALSE}, shouldPass: false},
	// tests OP_VERIFY (true). true is needed since else stack is empty.
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_VERIFY,
		btcscript.OP_TRUE}, shouldPass: true},
	// tests OP_VERIFY (false), will error out.
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_VERIFY,
		btcscript.OP_TRUE}, shouldPass: false},
	// tests OP_VERIFY with empty stack (errors)
	{script: []byte{btcscript.OP_VERIFY}, shouldPass: false},
	// test OP_RETURN immediately fails the script (empty stack)
	{script: []byte{btcscript.OP_RETURN}, shouldPass: false},
	// test OP_RETURN immediately fails the script (full stack)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_RETURN},
		shouldPass: false},
	// tests numequal with a trivial example (passing)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_NUMEQUAL}, shouldPass: true},
	// tests numequal with a trivial example (failing)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_NUMEQUAL}, shouldPass: false},
	// tests numequal with insufficient arguments (1/2)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_NUMEQUAL},
		shouldPass: false},
	// tests numequal with insufficient arguments (0/2)
	{script: []byte{btcscript.OP_NUMEQUAL}, shouldPass: false},
	// tests numnotequal with a trivial example (passing)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_NUMNOTEQUAL}, shouldPass: true},
	// tests numnotequal with a trivial example (failing)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_NUMNOTEQUAL}, shouldPass: false},
	// tests numnotequal with insufficient arguments (1/2)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_NUMNOTEQUAL},
		shouldPass: false},
	// tests numnotequal with insufficient arguments (0/2)
	{script: []byte{btcscript.OP_NUMNOTEQUAL}, shouldPass: false},
	// test numequal_verify with a trivial example (passing)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_NUMEQUALVERIFY, btcscript.OP_TRUE},
		shouldPass: true},
	// test numequal_verify with a trivial example (failing)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_NUMEQUALVERIFY, btcscript.OP_TRUE},
		shouldPass: false},
	// test OP_1ADD by adding 1 to 0
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_1ADD},
		shouldPass: true},
	// test OP_1ADD without args (should error)
	{script: []byte{btcscript.OP_1ADD}, shouldPass: false},
	// test OP_1NEGATE by adding 1 to -1
	{script: []byte{btcscript.OP_1NEGATE, btcscript.OP_1ADD},
		shouldPass: false},
	// test OP_1NEGATE by adding negating -1
	{script: []byte{btcscript.OP_1NEGATE, btcscript.OP_NEGATE},
		shouldPass: true},
	// test OP_NEGATE by adding 1 to -1
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_NEGATE,
		btcscript.OP_1ADD}, shouldPass: false},
	// test OP_NEGATE with no args
	{script: []byte{btcscript.OP_NEGATE}, shouldPass: false},
	// test OP_1SUB -> 1 - 1 = 0
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_1SUB},
		shouldPass: false},
	// test OP_1SUB -> negate(0 -1) = 1
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_1SUB,
		btcscript.OP_NEGATE}, shouldPass: true},
	// test OP_1SUB with empty stack
	{script: []byte{btcscript.OP_1SUB}, shouldPass: false},
	// OP_DEPTH with empty stack, means 0 on stack at end
	{script: []byte{btcscript.OP_DEPTH}, shouldPass: false},
	// 1 +1 -1 = 1. tests depth + add
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_DEPTH, btcscript.OP_ADD,
		btcscript.OP_1SUB}, shouldPass: true},
	// 1 +1 -1 = 0 . tests dept + add
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_DEPTH,
		btcscript.OP_ADD, btcscript.OP_1SUB, btcscript.OP_1SUB},
		shouldPass: false},
	// OP_ADD with only one thing on stack should error
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_ADD},
		shouldPass: false},
	// OP_ADD with nothing on stack should error
	{script: []byte{btcscript.OP_ADD}, shouldPass: false},
	// OP_SUB: 1-1=0
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_SUB}, shouldPass: false},
	// OP_SUB: 1+1-1=1
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_ADD, btcscript.OP_SUB}, shouldPass: true},
	// OP_SUB with only one thing on stack should error
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_SUB},
		shouldPass: false},
	// OP_SUB with nothing on stack should error
	{script: []byte{btcscript.OP_SUB}, shouldPass: false},
	// OP_LESSTHAN  1 < 1 == false
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_LESSTHAN}, shouldPass: false},
	// OP_LESSTHAN  1 < 0 == false
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_LESSTHAN}, shouldPass: false},
	// OP_LESSTHAN  0 < 1 == true
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_LESSTHAN}, shouldPass: true},
	// OP_LESSTHAN only one arg
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_LESSTHAN},
		shouldPass: false},
	// OP_LESSTHAN no args
	{script: []byte{btcscript.OP_LESSTHAN}, shouldPass: false},

	// OP_LESSTHANOREQUAL  1 <= 1 == true
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_LESSTHANOREQUAL}, shouldPass: true},
	// OP_LESSTHANOREQUAL  1 <= 0 == false
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_LESSTHANOREQUAL}, shouldPass: false},
	// OP_LESSTHANOREQUAL  0 <= 1 == true
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_LESSTHANOREQUAL}, shouldPass: true},
	// OP_LESSTHANOREQUAL only one arg
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_LESSTHANOREQUAL},
		shouldPass: false},
	// OP_LESSTHANOREQUAL no args
	{script: []byte{btcscript.OP_LESSTHANOREQUAL}, shouldPass: false},

	// OP_GREATERTHAN  1 > 1 == false
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_GREATERTHAN}, shouldPass: false},
	// OP_GREATERTHAN  1 > 0 == true
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_GREATERTHAN}, shouldPass: true},
	// OP_GREATERTHAN  0 > 1 == false
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_GREATERTHAN}, shouldPass: false},
	// OP_GREATERTHAN only one arg
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_GREATERTHAN},
		shouldPass: false},
	// OP_GREATERTHAN no args
	{script: []byte{btcscript.OP_GREATERTHAN}, shouldPass: false},

	// OP_GREATERTHANOREQUAL  1 >= 1 == true
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_GREATERTHANOREQUAL}, shouldPass: true},
	// OP_GREATERTHANOREQUAL  1 >= 0 == false
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_GREATERTHANOREQUAL}, shouldPass: true},
	// OP_GREATERTHANOREQUAL  0 >= 1 == true
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_GREATERTHANOREQUAL}, shouldPass: false},
	// OP_GREATERTHANOREQUAL only one arg
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_GREATERTHANOREQUAL},
		shouldPass: false},
	// OP_GREATERTHANOREQUAL no args
	{script: []byte{btcscript.OP_GREATERTHANOREQUAL}, shouldPass: false},

	// OP_MIN basic functionality -> min(0,1) = 0 = min(1,0)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_MIN}, shouldPass: false},
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_MIN}, shouldPass: false},
	// OP_MIN -> 1 arg errors
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_MIN},
		shouldPass: false},
	// OP_MIN -> 0 arg errors
	{script: []byte{btcscript.OP_MIN}, shouldPass: false},
	// OP_MAX basic functionality -> max(0,1) = 1 = max(1,0)
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_MAX}, shouldPass: true},
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_MAX}, shouldPass: true},
	// OP_MAX -> 1 arg errors
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_MAX},
		shouldPass: false},
	// OP_MAX -> 0 arg errors
	{script: []byte{btcscript.OP_MAX}, shouldPass: false},

	// By this point we know a number of operations appear to be working
	// correctly. we can use them to test the other number pushing
	// operations
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_1ADD, btcscript.OP_2,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_2, btcscript.OP_1ADD, btcscript.OP_3,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_3, btcscript.OP_1ADD, btcscript.OP_4,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_4, btcscript.OP_1ADD, btcscript.OP_5,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_5, btcscript.OP_1ADD, btcscript.OP_6,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_6, btcscript.OP_1ADD, btcscript.OP_7,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_7, btcscript.OP_1ADD, btcscript.OP_8,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_8, btcscript.OP_1ADD, btcscript.OP_9,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_9, btcscript.OP_1ADD, btcscript.OP_10,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_10, btcscript.OP_1ADD, btcscript.OP_11,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_11, btcscript.OP_1ADD, btcscript.OP_12,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_12, btcscript.OP_1ADD, btcscript.OP_13,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_13, btcscript.OP_1ADD, btcscript.OP_14,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_14, btcscript.OP_1ADD, btcscript.OP_15,
		btcscript.OP_EQUAL}, shouldPass: true},
	{script: []byte{btcscript.OP_15, btcscript.OP_1ADD, btcscript.OP_16,
		btcscript.OP_EQUAL}, shouldPass: true},

	// Test OP_WITHIN x, min, max
	// 0 <= 1 < 2
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE, btcscript.OP_2,
		btcscript.OP_WITHIN}, shouldPass: true},
	// 1 <= 0 < 2 FAIL
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE, btcscript.OP_2,
		btcscript.OP_WITHIN}, shouldPass: false},
	// 1 <= 1 < 2
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE, btcscript.OP_2,
		btcscript.OP_WITHIN}, shouldPass: true},
	// 1 <= 2 < 2 FAIL
	{script: []byte{btcscript.OP_2, btcscript.OP_TRUE, btcscript.OP_2,
		btcscript.OP_WITHIN}, shouldPass: false},
	// only two arguments
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_WITHIN}, shouldPass: false},
	// only one argument
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_WITHIN},
		shouldPass: false},
	// no arguments
	{script: []byte{btcscript.OP_WITHIN}, shouldPass: false},

	// OP_BOOLAND
	// 1 && 1 == 1
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_BOOLAND}, shouldPass: true},
	// 1 && 0 == 0
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_BOOLAND}, shouldPass: false},
	// 0 && 1 == 0
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_BOOLAND}, shouldPass: false},
	// 0 && 0 == 0
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_FALSE,
		btcscript.OP_BOOLAND}, shouldPass: false},
	// 0 && <nothing> - boom
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_BOOLAND},
		shouldPass: false},
	// <nothing> && <nothing> - boom
	{script: []byte{btcscript.OP_BOOLAND}, shouldPass: false},

	// OP_BOOLOR
	// 1 || 1 == 1
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_TRUE,
		btcscript.OP_BOOLOR}, shouldPass: true},
	// 1 || 0 == 1
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_FALSE,
		btcscript.OP_BOOLOR}, shouldPass: true},
	// 0 || 1 == 1
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_TRUE,
		btcscript.OP_BOOLOR}, shouldPass: true},
	// 0 || 0 == 0
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_FALSE,
		btcscript.OP_BOOLOR}, shouldPass: false},
	// 0 && <nothing> - boom
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_BOOLOR},
		shouldPass: false},
	// <nothing> && <nothing> - boom
	{script: []byte{btcscript.OP_BOOLOR}, shouldPass: false},

	// OP_0NOTEQUAL
	//  1 with input != 0 XXX check output is actually 1.
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_2, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_3, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_4, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_5, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_6, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_7, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_8, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_9, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_10, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_11, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_12, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_13, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_14, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_15, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_16, btcscript.OP_0NOTEQUAL},
		shouldPass: true},
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_0NOTEQUAL}, shouldPass: false},
	// No arguments also blows up
	{script: []byte{btcscript.OP_0NOTEQUAL}, shouldPass: false},

	// OP_NOT: 1 i input is 0, else 0
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_2, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_3, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_4, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_5, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_6, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_7, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_8, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_9, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_10, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_11, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_12, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_13, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_14, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_15, btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_16, btcscript.OP_NOT}, shouldPass: false},
	// check negative numbers too
	{script: []byte{btcscript.OP_TRUE, btcscript.OP_NEGATE,
		btcscript.OP_NOT}, shouldPass: false},
	{script: []byte{btcscript.OP_FALSE, btcscript.OP_NOT},
		shouldPass: true},
	// No arguments also blows up
	{script: []byte{btcscript.OP_NOT}, shouldPass: false},

	// Conditional Execution
	{script: []byte{btcscript.OP_0, btcscript.OP_IF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_2, btcscript.OP_ENDIF}, shouldPass: true},
	{script: []byte{btcscript.OP_1, btcscript.OP_IF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_2, btcscript.OP_ENDIF}, shouldPass: false},
	{script: []byte{btcscript.OP_1, btcscript.OP_NOTIF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_2, btcscript.OP_ENDIF}, shouldPass: true},
	{script: []byte{btcscript.OP_0, btcscript.OP_NOTIF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_2, btcscript.OP_ENDIF}, shouldPass: false},
	{script: []byte{btcscript.OP_0, btcscript.OP_IF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_2}, shouldFail: btcscript.StackErrMissingEndif},
	{script: []byte{btcscript.OP_1, btcscript.OP_NOTIF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_2}, shouldFail: btcscript.StackErrMissingEndif},
	{script: []byte{btcscript.OP_1, btcscript.OP_1, btcscript.OP_IF, btcscript.OP_IF, btcscript.OP_1, btcscript.OP_ELSE, btcscript.OP_0, btcscript.OP_ENDIF, btcscript.OP_ENDIF}, shouldPass: true},
	{script: []byte{btcscript.OP_1, btcscript.OP_IF, btcscript.OP_IF, btcscript.OP_1, btcscript.OP_ELSE, btcscript.OP_0, btcscript.OP_ENDIF, btcscript.OP_ENDIF}, shouldFail: btcscript.StackErrUnderflow},
	{script: []byte{btcscript.OP_0, btcscript.OP_IF, btcscript.OP_IF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_0, btcscript.OP_ENDIF, btcscript.OP_ELSE, btcscript.OP_1, btcscript.OP_ENDIF}, shouldPass: true},
	{script: []byte{btcscript.OP_0, btcscript.OP_IF, btcscript.OP_NOTIF, btcscript.OP_0, btcscript.OP_ELSE, btcscript.OP_0, btcscript.OP_ENDIF, btcscript.OP_ELSE, btcscript.OP_1, btcscript.OP_ENDIF}, shouldPass: true},
	{script: []byte{btcscript.OP_NOTIF, btcscript.OP_0, btcscript.OP_ENDIF}, shouldFail: btcscript.StackErrUnderflow},
	{script: []byte{btcscript.OP_ELSE, btcscript.OP_0, btcscript.OP_ENDIF}, shouldFail: btcscript.StackErrNoIf},
	{script: []byte{btcscript.OP_ENDIF}, shouldFail: btcscript.StackErrNoIf},
	/* up here because error from sig parsing is undefined. */
	{script: []byte{btcscript.OP_1, btcscript.OP_1, btcscript.OP_DATA_65,
		0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b, 0x13,
		0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7, 0x15, 0x9b,
		0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d, 0x71, 0x30, 0x2f,
		0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3, 0x73, 0x97, 0xf5, 0x54,
		0xa7, 0xdf, 0x5f, 0x14, 0x2c, 0x21, 0xc1, 0xb7, 0x30, 0x3b,
		0x8a, 0x06, 0x26, 0xf1, 0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70,
		0x4f, 0x7e, 0x6c, 0xd8, 0x4c,
		btcscript.OP_1, btcscript.OP_CHECK_MULTISIG},
		shouldPass: false},
	/* up here because no defined error case. */
	{script: []byte{btcscript.OP_1, btcscript.OP_1, btcscript.OP_DATA_65,
		0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b, 0x13,
		0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7, 0x15, 0x9b,
		0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d, 0x71, 0x30, 0x2f,
		0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3, 0x73, 0x97, 0xf5, 0x54,
		0xa7, 0xdf, 0x5f, 0x14, 0x2c, 0x21, 0xc1, 0xb7, 0x30, 0x3b,
		0x8a, 0x06, 0x26, 0xf1, 0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70,
		0x4f, 0x7e, 0x6c, 0xd8, 0x4c,
		btcscript.OP_1, btcscript.OP_CHECKMULTISIGVERIFY},
		shouldPass: false},

	// Invalid Opcodes
	{script: []byte{186}, shouldPass: false},
	{script: []byte{187}, shouldPass: false},
	{script: []byte{188}, shouldPass: false},
	{script: []byte{189}, shouldPass: false},
	{script: []byte{190}, shouldPass: false},
	{script: []byte{191}, shouldPass: false},
	{script: []byte{192}, shouldPass: false},
	{script: []byte{193}, shouldPass: false},
	{script: []byte{194}, shouldPass: false},
	{script: []byte{195}, shouldPass: false},
	{script: []byte{195}, shouldPass: false},
	{script: []byte{196}, shouldPass: false},
	{script: []byte{197}, shouldPass: false},
	{script: []byte{198}, shouldPass: false},
	{script: []byte{199}, shouldPass: false},
	{script: []byte{200}, shouldPass: false},
	{script: []byte{201}, shouldPass: false},
	{script: []byte{202}, shouldPass: false},
	{script: []byte{203}, shouldPass: false},
	{script: []byte{204}, shouldPass: false},
	{script: []byte{205}, shouldPass: false},
	{script: []byte{206}, shouldPass: false},
	{script: []byte{207}, shouldPass: false},
	{script: []byte{208}, shouldPass: false},
	{script: []byte{209}, shouldPass: false},
	{script: []byte{210}, shouldPass: false},
	{script: []byte{211}, shouldPass: false},
	{script: []byte{212}, shouldPass: false},
	{script: []byte{213}, shouldPass: false},
	{script: []byte{214}, shouldPass: false},
	{script: []byte{215}, shouldPass: false},
	{script: []byte{216}, shouldPass: false},
	{script: []byte{217}, shouldPass: false},
	{script: []byte{218}, shouldPass: false},
	{script: []byte{219}, shouldPass: false},
	{script: []byte{220}, shouldPass: false},
	{script: []byte{221}, shouldPass: false},
	{script: []byte{222}, shouldPass: false},
	{script: []byte{223}, shouldPass: false},
	{script: []byte{224}, shouldPass: false},
	{script: []byte{225}, shouldPass: false},
	{script: []byte{226}, shouldPass: false},
	{script: []byte{227}, shouldPass: false},
	{script: []byte{228}, shouldPass: false},
	{script: []byte{229}, shouldPass: false},
	{script: []byte{230}, shouldPass: false},
	{script: []byte{231}, shouldPass: false},
	{script: []byte{232}, shouldPass: false},
	{script: []byte{233}, shouldPass: false},
	{script: []byte{234}, shouldPass: false},
	{script: []byte{235}, shouldPass: false},
	{script: []byte{236}, shouldPass: false},
	{script: []byte{237}, shouldPass: false},
	{script: []byte{238}, shouldPass: false},
	{script: []byte{239}, shouldPass: false},
	{script: []byte{240}, shouldPass: false},
	{script: []byte{241}, shouldPass: false},
	{script: []byte{242}, shouldPass: false},
	{script: []byte{243}, shouldPass: false},
	{script: []byte{244}, shouldPass: false},
	{script: []byte{245}, shouldPass: false},
	{script: []byte{246}, shouldPass: false},
	{script: []byte{247}, shouldPass: false},
	{script: []byte{248}, shouldPass: false},
	{script: []byte{249}, shouldPass: false},
	{script: []byte{250}, shouldPass: false},
	{script: []byte{251}, shouldPass: false},
	{script: []byte{252}, shouldPass: false},
}

func testScript(t *testing.T, script []byte) (err error) {
	// mock up fake tx.
	tx := &btcwire.MsgTx{
		Version: 1,
		TxIn: []*btcwire.TxIn{
			&btcwire.TxIn{
				PreviousOutpoint: btcwire.OutPoint{
					Hash:  btcwire.ShaHash{},
					Index: 0xffffffff,
				},
				SignatureScript: []byte{btcscript.OP_NOP},
				Sequence:        0xffffffff,
			},
		},
		TxOut: []*btcwire.TxOut{
			&btcwire.TxOut{
				Value:    0x12a05f200,
				PkScript: []byte{},
			},
		},
		LockTime: 0,
	}

	tx.TxOut[0].PkScript = script

	engine, err := btcscript.NewScript(tx.TxIn[0].SignatureScript,
		tx.TxOut[0].PkScript, 0, tx, 1, false)
	if err != nil {
		return err
	}
	return engine.Execute()
}

func TestScripts(t *testing.T) {
	log, err := seelog.LoggerFromWriterWithMinLevel(os.Stdout,
		seelog.InfoLvl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v", err)
		return
	}
	defer log.Flush()
	btcscript.UseLogger(log)
	// for each entry in the list
	for i := range opcodeTests {
		shouldPass := opcodeTests[i].shouldPass
		shouldFail := opcodeTests[i].shouldFail
		err := testScript(t, opcodeTests[i].script)
		if shouldFail != nil {
			if err == nil {
				t.Errorf("test %d passed should fail with %v", i, err)
			} else if shouldFail != err {
				t.Errorf("test %d failed with wrong error [%v], expected [%v]", i, err, shouldFail)
			}
		}
		if shouldPass && err != nil {
			t.Errorf("test %d failed: %v", i, err)
		} else if !shouldPass && err == nil {
			t.Errorf("test %d passed, should fail", i)
		}
	}
}

// Detailed tests for opcodes, we inspect machine state before and after the
// opcode and check that it has the effect on the state that we expect.
type detailedTest struct {
	name           string
	before         [][]byte
	altbefore      [][]byte
	script         []byte
	expectedReturn error
	after          [][]byte
	altafter       [][]byte
	disassembly    string
	disassemblyerr error
}

var detailedTests = []detailedTest{
	{
		name:        "noop",
		before:      [][]byte{{1}, {2}, {3}, {4}, {5}},
		script:      []byte{btcscript.OP_NOP},
		after:       [][]byte{{1}, {2}, {3}, {4}, {5}},
		disassembly: "OP_NOP",
	},
	{
		name:        "dup",
		before:      [][]byte{{1}},
		script:      []byte{btcscript.OP_DUP},
		after:       [][]byte{{1}, {1}},
		disassembly: "OP_DUP",
	},
	{
		name:        "dup2",
		before:      [][]byte{{1}, {2}},
		script:      []byte{btcscript.OP_2DUP},
		after:       [][]byte{{1}, {2}, {1}, {2}},
		disassembly: "OP_2DUP",
	},
	{
		name:        "dup3",
		before:      [][]byte{{1}, {2}, {3}},
		script:      []byte{btcscript.OP_3DUP},
		after:       [][]byte{{1}, {2}, {3}, {1}, {2}, {3}},
		disassembly: "OP_3DUP",
	},
	{
		name:           "dup too much",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_DUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_DUP",
	},
	{
		name:           "2dup too much",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_2DUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_2DUP",
	},
	{
		name:           "2dup way too much",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_2DUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_2DUP",
	},
	{
		name:           "3dup too much",
		before:         [][]byte{{1}, {2}},
		script:         []byte{btcscript.OP_3DUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_3DUP",
	},
	{
		name:           "3dup kinda too much",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_3DUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_3DUP",
	},
	{
		name:           "3dup way too much",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_3DUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_3DUP",
	},
	{
		name:        "Nip",
		before:      [][]byte{{1}, {2}, {3}},
		script:      []byte{btcscript.OP_NIP},
		after:       [][]byte{{1}, {3}},
		disassembly: "OP_NIP",
	},
	{
		name:           "Nip too much",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_NIP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{{2}, {3}},
		disassembly:    "OP_NIP",
	},
	{
		name:        "keep on tucking",
		before:      [][]byte{{1}, {2}, {3}},
		script:      []byte{btcscript.OP_TUCK},
		after:       [][]byte{{1}, {3}, {2}, {3}},
		disassembly: "OP_TUCK",
	},
	{
		name:           "a little tucked up",
		before:         [][]byte{{1}}, // too few arguments for tuck
		script:         []byte{btcscript.OP_TUCK},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{},
		disassembly:    "OP_TUCK",
	},
	{
		name:           "all tucked up",
		before:         [][]byte{}, // too few arguments  for tuck
		script:         []byte{btcscript.OP_TUCK},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_TUCK",
	},
	{
		name:        "drop 1",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_DROP},
		after:       [][]byte{{1}, {2}, {3}},
		disassembly: "OP_DROP",
	},
	{
		name:        "drop 2",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_2DROP},
		after:       [][]byte{{1}, {2}},
		disassembly: "OP_2DROP",
	},
	{
		name:           "drop too much",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_DROP},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_DROP",
	},
	{
		name:           "2drop too much",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_2DROP},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_2DROP",
	},
	{
		name:           "2drop far too much",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_2DROP},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_2DROP",
	},
	{
		name:        "Rot1",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_ROT},
		after:       [][]byte{{1}, {3}, {4}, {2}},
		disassembly: "OP_ROT",
	},
	{
		name:        "Rot2",
		before:      [][]byte{{1}, {2}, {3}, {4}, {5}, {6}},
		script:      []byte{btcscript.OP_2ROT},
		after:       [][]byte{{3}, {4}, {5}, {6}, {1}, {2}},
		disassembly: "OP_2ROT",
	},
	{
		name:           "Rot too little",
		before:         [][]byte{{1}, {2}},
		script:         []byte{btcscript.OP_ROT},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_ROT",
	},
	{
		name:        "Swap1",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_SWAP},
		after:       [][]byte{{1}, {2}, {4}, {3}},
		disassembly: "OP_SWAP",
	},
	{
		name:        "Swap2",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_2SWAP},
		after:       [][]byte{{3}, {4}, {1}, {2}},
		disassembly: "OP_2SWAP",
	},
	{
		name:           "Swap too little",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_SWAP},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_SWAP",
	},
	{
		name:        "Over1",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_OVER},
		after:       [][]byte{{1}, {2}, {3}, {4}, {3}},
		disassembly: "OP_OVER",
	},
	{
		name:        "Over2",
		before:      [][]byte{{1}, {2}, {3}, {4}},
		script:      []byte{btcscript.OP_2OVER},
		after:       [][]byte{{1}, {2}, {3}, {4}, {1}, {2}},
		disassembly: "OP_2OVER",
	},
	{
		name:           "Over too little",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_OVER},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_OVER",
	},
	{
		name:        "Pick1",
		before:      [][]byte{{1}, {2}, {3}, {4}, {1}},
		script:      []byte{btcscript.OP_PICK},
		after:       [][]byte{{1}, {2}, {3}, {4}, {3}},
		disassembly: "OP_PICK",
	},
	{
		name:        "Pick2",
		before:      [][]byte{{1}, {2}, {3}, {4}, {2}},
		script:      []byte{btcscript.OP_PICK},
		after:       [][]byte{{1}, {2}, {3}, {4}, {2}},
		disassembly: "OP_PICK",
	},
	{
		name:           "Pick too little",
		before:         [][]byte{{1}, {1}},
		script:         []byte{btcscript.OP_PICK},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_PICK",
	},
	{
		name:           "Pick nothing",
		before:         [][]byte{{}},
		script:         []byte{btcscript.OP_PICK},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_PICK",
	},
	{
		name:           "Pick no args",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_PICK},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_PICK",
	},
	{
		name:           "Pick stupid numbers",
		before:         [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		script:         []byte{btcscript.OP_PICK},
		expectedReturn: btcscript.StackErrNumberTooBig,
		disassembly:    "OP_PICK",
	},
	{
		name:        "Roll1",
		before:      [][]byte{{1}, {2}, {3}, {4}, {1}},
		script:      []byte{btcscript.OP_ROLL},
		after:       [][]byte{{1}, {2}, {4}, {3}},
		disassembly: "OP_ROLL",
	},
	{
		name:        "Roll2",
		before:      [][]byte{{1}, {2}, {3}, {4}, {2}},
		script:      []byte{btcscript.OP_ROLL},
		after:       [][]byte{{1}, {3}, {4}, {2}},
		disassembly: "OP_ROLL",
	},
	{
		name:           "Roll too little",
		before:         [][]byte{{1}, {1}},
		script:         []byte{btcscript.OP_ROLL},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_ROLL",
	},
	{
		name:           "Roll nothing ",
		before:         [][]byte{{1}},
		script:         []byte{btcscript.OP_ROLL},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_ROLL",
	},
	{
		name:           "Roll no args ",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_ROLL},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_ROLL",
	},
	{
		name:           "Roll stupid numbers",
		before:         [][]byte{{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		script:         []byte{btcscript.OP_ROLL},
		expectedReturn: btcscript.StackErrNumberTooBig,
		disassembly:    "OP_ROLL",
	},
	{
		name:        "ifdup (positive)",
		before:      [][]byte{{1}},
		script:      []byte{btcscript.OP_IFDUP},
		after:       [][]byte{{1}, {1}},
		disassembly: "OP_IFDUP",
	},
	{
		name:        "ifdup (negative)",
		before:      [][]byte{{0}},
		script:      []byte{btcscript.OP_IFDUP},
		after:       [][]byte{{0}},
		disassembly: "OP_IFDUP",
	},
	{
		name:           "ifdup (empty)",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_IFDUP},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{{0}},
		disassembly:    "OP_IFDUP",
	},
	{
		name:        "toaltastack",
		before:      [][]byte{{1}},
		altbefore:   [][]byte{},
		script:      []byte{btcscript.OP_TOALTSTACK},
		after:       [][]byte{},
		altafter:    [][]byte{{1}},
		disassembly: "OP_TOALTSTACK",
	},
	{
		name:           "toaltastack (empty)",
		before:         [][]byte{},
		altbefore:      [][]byte{},
		script:         []byte{btcscript.OP_TOALTSTACK},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_TOALTSTACK",
	},
	{
		name:        "fromaltastack",
		before:      [][]byte{},
		altbefore:   [][]byte{{1}},
		script:      []byte{btcscript.OP_FROMALTSTACK},
		after:       [][]byte{{1}},
		altafter:    [][]byte{},
		disassembly: "OP_FROMALTSTACK",
	},
	{
		name:           "fromaltastack (empty)",
		before:         [][]byte{},
		altbefore:      [][]byte{},
		script:         []byte{btcscript.OP_FROMALTSTACK},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_FROMALTSTACK",
	},
	{
		name:        "op_size (1)",
		before:      [][]byte{{1}},
		script:      []byte{btcscript.OP_SIZE},
		after:       [][]byte{{1}, {1}},
		disassembly: "OP_SIZE",
	},
	{
		name:        "op_size (5)",
		before:      [][]byte{{1, 2, 3, 4, 5}},
		script:      []byte{btcscript.OP_SIZE},
		after:       [][]byte{{1, 2, 3, 4, 5}, {5}},
		disassembly: "OP_SIZE",
	},
	{
		name:   "op_size (0)",
		before: [][]byte{{}},
		script: []byte{btcscript.OP_SIZE},
		// pushInt(0) actually gives an empty array, still counts as 0
		after:       [][]byte{{}, {}},
		disassembly: "OP_SIZE",
	},
	{
		name:           "op_size (invalid)",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_SIZE},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_SIZE",
	},
	{
		name:        "OP_EQUAL (valid)",
		before:      [][]byte{{1, 2, 3, 4}, {1, 2, 3, 4}},
		script:      []byte{btcscript.OP_EQUAL},
		after:       [][]byte{{1}},
		disassembly: "OP_EQUAL",
	},
	{
		name:        "OP_EQUAL (invalid)",
		before:      [][]byte{{1, 2, 3, 4}, {1, 2, 3, 3}},
		script:      []byte{btcscript.OP_EQUAL},
		after:       [][]byte{{0}},
		disassembly: "OP_EQUAL",
	},
	{
		name:           "OP_EQUAL (one arg)",
		before:         [][]byte{{1, 2, 3, 4}},
		script:         []byte{btcscript.OP_EQUAL},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{{0}},
		disassembly:    "OP_EQUAL",
	},
	{
		name:           "OP_EQUAL (no arg)",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_EQUAL},
		expectedReturn: btcscript.StackErrUnderflow,
		after:          [][]byte{{0}},
		disassembly:    "OP_EQUAL",
	},
	{
		name:        "OP_EQUALVERIFY (valid)",
		before:      [][]byte{{1, 2, 3, 4}, {1, 2, 3, 4}},
		script:      []byte{btcscript.OP_EQUALVERIFY},
		after:       [][]byte{},
		disassembly: "OP_EQUALVERIFY",
	},
	{
		name:           "OP_EQUALVERIFY (invalid)",
		before:         [][]byte{{1, 2, 3, 4}, {1, 2, 3, 3}},
		script:         []byte{btcscript.OP_EQUALVERIFY},
		expectedReturn: btcscript.StackErrVerifyFailed,
		after:          [][]byte{},
		disassembly:    "OP_EQUALVERIFY",
	},
	{
		name:           "OP_EQUALVERIFY (one arg)",
		before:         [][]byte{{1, 2, 3, 4}},
		script:         []byte{btcscript.OP_EQUALVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_EQUALVERIFY",
	},
	{
		name:           "OP_EQUALVERIFY (no arg)",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_EQUALVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_EQUALVERIFY",
	},
	{
		name:        "OP_1NEGATE",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_1NEGATE},
		after:       [][]byte{{0x81}},
		disassembly: "OP_1NEGATE",
	},
	{
		name:        "add one to minus one",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_1NEGATE, btcscript.OP_1ADD},
		after:       [][]byte{{}}, // 0
		disassembly: "OP_1NEGATE OP_1ADD",
	},
	{
		name:        "OP_ABS (positive)",
		before:      [][]byte{{1}},
		script:      []byte{btcscript.OP_ABS},
		after:       [][]byte{{1}},
		disassembly: "OP_ABS",
	},
	{
		name:        "OP_ABS (negative)",
		before:      [][]byte{{0x81}},
		script:      []byte{btcscript.OP_ABS},
		after:       [][]byte{{1}},
		disassembly: "OP_ABS",
	},
	{
		name:           "OP_ABS (empty)",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_ABS},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_ABS",
	},
	{
		name:        "op_data_1",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_1, 1},
		after:       [][]byte{{1}},
		disassembly: "01",
	},
	{
		name:        "op_data_2",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_2, 1, 2},
		after:       [][]byte{{1, 2}},
		disassembly: "0102",
	},
	{
		name:        "op_data_3",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_3, 1, 2, 3},
		after:       [][]byte{{1, 2, 3}},
		disassembly: "010203",
	},
	{
		name:        "op_data_4",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_4, 1, 2, 3, 4},
		after:       [][]byte{{1, 2, 3, 4}},
		disassembly: "01020304",
	},
	{
		name:        "op_data_5",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_5, 1, 2, 3, 4, 5},
		after:       [][]byte{{1, 2, 3, 4, 5}},
		disassembly: "0102030405",
	},
	{
		name:        "op_data_6",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_6, 1, 2, 3, 4, 5, 6},
		after:       [][]byte{{1, 2, 3, 4, 5, 6}},
		disassembly: "010203040506",
	},
	{
		name:        "op_data_7",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_7, 1, 2, 3, 4, 5, 6, 7},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7}},
		disassembly: "01020304050607",
	},
	{
		name:        "op_data_8",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_8, 1, 2, 3, 4, 5, 6, 7, 8},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8}},
		disassembly: "0102030405060708",
	},
	{
		name:        "op_data_9",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_9, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		disassembly: "010203040506070809",
	},
	{
		name:        "op_data_10",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_DATA_10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		disassembly: "0102030405060708090a",
	},
	{
		name:   "op_data_11",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_11, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		disassembly: "0102030405060708090a0b",
	},
	{
		name:   "op_data_12",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_12, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		disassembly: "0102030405060708090a0b0c",
	},
	{
		name:   "op_data_13",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_13, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}},
		disassembly: "0102030405060708090a0b0c0d",
	},
	{
		name:   "op_data_14",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_14, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14},
		after:       [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}},
		disassembly: "0102030405060708090a0b0c0d0e",
	},
	{
		name:   "op_data_15",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_15, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15}},
		disassembly: "0102030405060708090a0b0c0d0e0f",
	},
	{
		name:   "op_data_16",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_16, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16}},
		disassembly: "0102030405060708090a0b0c0d0e0f10",
	},
	{
		name:   "op_data_17",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_17, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17}},
		disassembly: "0102030405060708090a0b0c0d0e0f1011",
	},
	{
		name:   "op_data_18",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_18, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112",
	},
	{
		name:   "op_data_19",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_19, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19}},
		disassembly: "0102030405060708090a0b0c0d0e0f10111213",
	},
	{
		name:   "op_data_20",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_20, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20}},
		disassembly: "0102030405060708090a0b0c0d0e0f1011121314",
	},
	{
		name:   "op_data_21",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_21, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415",
	},
	{
		name:   "op_data_22",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_22, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22}},
		disassembly: "0102030405060708090a0b0c0d0e0f10111213141516",
	},
	{
		name:   "op_data_23",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_23, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23}},
		disassembly: "0102030405060708090a0b0c0d0e0f1011121314151617",
	},
	{
		name:   "op_data_24",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_24, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718",
	},
	{
		name:   "op_data_25",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_25, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}},
		disassembly: "0102030405060708090a0b0c0d0e0f10111213141516171819",
	},
	{
		name:   "op_data_26",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_26, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a",
	},
	{
		name:   "op_data_27",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_27, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b",
	},
	{
		name:   "op_data_28",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_28, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c",
	},
	{
		name:   "op_data_29",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_29, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d",
	},
	{
		name:   "op_data_30",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_30, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e",
	},
	{
		name:   "op_data_31",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_31, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	},
	{
		name:   "op_data_32",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_32, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
	},
	{
		name:   "op_data_33",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_33, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f2021",
	},
	{
		name:   "op_data_34",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_34, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122",
	},
	{
		name:   "op_data_35",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_35, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20212223",
	},
	{
		name:   "op_data_36",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_36, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f2021222324",
	},
	{
		name:   "op_data_37",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_37, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425",
	},
	{
		name:   "op_data_38",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_38, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20212223242526",
	},
	{
		name:   "op_data_39",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_39, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f2021222324252627",
	},
	{
		name:   "op_data_40",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_40, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728",
	},
	{
		name:   "op_data_41",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_41, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20212223242526272829",
	},
	{
		name:   "op_data_42",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_42, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a",
	},
	{
		name:   "op_data_43",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_43, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b",
	},
	{
		name:   "op_data_44",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_44, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c",
	},
	{
		name:   "op_data_45",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_45, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d",
	},
	{
		name:   "op_data_46",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_46, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e",
	},
	{
		name:   "op_data_47",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_47, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f",
	},
	{
		name:   "op_data_48",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_48, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f30",
	},
	{
		name:   "op_data_49",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_49, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031",
	},
	{
		name:   "op_data_50",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_50, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132",
	},
	{
		name:   "op_data_51",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_51, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f30313233",
	},
	{
		name:   "op_data_52",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_52, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334",
	},
	{
		name:   "op_data_53",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_53, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435",
	},
	{
		name:   "op_data_54",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_54, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f30313233343536",
	},
	{
		name:   "op_data_55",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_55, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637",
	},
	{
		name:   "op_data_56",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_56, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738",
	},
	{
		name:   "op_data_57",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_57, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f30313233343536373839",
	},
	{
		name:   "op_data_58",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_58, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a",
	},
	{
		name:   "op_data_59",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_59, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b",
	},
	{
		name:   "op_data_60",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_60, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c",
	},
	{
		name:   "op_data_61",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_61, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d",
	},
	{
		name:   "op_data_62",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_62, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e",
	},
	{
		name:   "op_data_63",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_63, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f",
	},
	{
		name:   "op_data_64",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_64, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40",
	},
	{
		name:   "op_data_65",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_65, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f4041",
	},
	{
		name:   "op_data_66",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_66, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142",
	},
	{
		name:   "op_data_67",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_67, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40414243",
	},
	{
		name:   "op_data_68",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_68, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f4041424344",
	},
	{
		name:   "op_data_69",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_69, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445",
	},
	{
		name:   "op_data_70",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_70, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69, 70}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40414243444546",
	},
	{
		name:   "op_data_71",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_71, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69, 70, 71}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f4041424344454647",
	},
	{
		name:   "op_data_72",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_72, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71, 72},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69, 70, 71, 72}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748",
	},
	{
		name:   "op_data_73",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_73, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71, 72, 73},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40414243444546474849",
	},
	{
		name:   "op_data_74",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_74, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71, 72, 73, 74},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74,
		}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a",
	},
	{
		name:   "op_data_75",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_75, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
			22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33,
			34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
			46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
			58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71, 72, 73, 74, 75},
		after: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
			15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
			27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
			39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
			63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74,
			75}},
		disassembly: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b",
	},
	{
		name:           "op_data too short",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_DATA_2, 1},
		expectedReturn: btcscript.StackErrShortScript,
		disassemblyerr: btcscript.StackErrShortScript,
	},
	{
		name:        "op_pushdata_1",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_PUSHDATA1, 1, 2},
		after:       [][]byte{{2}},
		disassembly: "02",
	},
	{
		name:           "op_pushdata_1 too short",
		script:         []byte{btcscript.OP_PUSHDATA1, 1},
		expectedReturn: btcscript.StackErrShortScript,
		disassemblyerr: btcscript.StackErrShortScript,
	},
	{
		name:        "op_pushdata_2",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_PUSHDATA2, 2, 0, 2, 4},
		after:       [][]byte{{2, 4}},
		disassembly: "0204",
	},
	{
		name:           "op_pushdata_2 too short",
		script:         []byte{btcscript.OP_PUSHDATA2, 2, 0},
		expectedReturn: btcscript.StackErrShortScript,
		disassemblyerr: btcscript.StackErrShortScript,
	},
	{
		name:        "op_pushdata_4",
		before:      [][]byte{},
		script:      []byte{btcscript.OP_PUSHDATA4, 4, 0, 0, 0, 2, 4, 8, 16},
		after:       [][]byte{{2, 4, 8, 16}},
		disassembly: "02040810",
	},
	{
		name:           "op_pushdata_4 too short",
		script:         []byte{btcscript.OP_PUSHDATA4, 4, 0, 0, 0},
		expectedReturn: btcscript.StackErrShortScript,
		disassemblyerr: btcscript.StackErrShortScript,
	},
	// XXX also pushdata cases where the pushed data isn't long enough,
	// no real error type defined for that as of yet.

	{
		name:           "OP_SHA1 no args",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_SHA1},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_SHA1",
	},
	{
		name:           "OP_SHA256 no args",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_SHA256},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_SHA256",
	},
	{
		name:           "OP_RIPEMD160 no args",
		before:         [][]byte{},
		script:         []byte{btcscript.OP_RIPEMD160},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_RIPEMD160",
	},
	// data taken from transaction
	// 4cbb6924e5f9788d7fcf0a1ce8c175bf9befa43eb5e23386b69bc4dce49da71c
	// in block 103307
	// First do it in component parts to make sure the sha256 and ripemd160
	// opcodes work
	{
		name: "op_hash160 the hard way",
		before: [][]byte{{0x04, 0x0f, 0xa4, 0x92, 0xe3, 0x59, 0xde, 0xe8, 0x4b,
			0x53, 0xfe, 0xc5, 0xe9, 0x18, 0xb7, 0xfd, 0x62, 0x1e,
			0xb7, 0xe5, 0x63, 0x38, 0xc5, 0xfb, 0xff, 0x71, 0xd9,
			0x1d, 0x17, 0x22, 0xda, 0x58, 0xf1, 0x0f, 0x9e, 0x8f,
			0x41, 0x2f, 0x39, 0x9c, 0xb3, 0x06, 0x70, 0xa7, 0x27,
			0xe9, 0x91, 0x94, 0xaa, 0x69, 0x27, 0xaf, 0xf2, 0x54,
			0x16, 0xec, 0x48, 0x9d, 0x45, 0x3a, 0x80, 0x7e, 0x03,
			0xc0, 0x83}},
		script: []byte{btcscript.OP_SHA256, btcscript.OP_RIPEMD160},
		after: [][]byte{{0x8b, 0xfa, 0x5c, 0x1f, 0x68, 0x5f, 0x13, 0x86, 0x3e,
			0x74, 0x2e, 0x1b, 0xaf, 0x15, 0xf1, 0x71, 0xad, 0x49,
			0x8b, 0x8f}},
		disassembly: "OP_SHA256 OP_RIPEMD160",
	},
	// Then test it the ``normal'' way.
	{
		name: "op_hash160",
		before: [][]byte{{0x04, 0x0f, 0xa4, 0x92, 0xe3, 0x59, 0xde, 0xe8, 0x4b,
			0x53, 0xfe, 0xc5, 0xe9, 0x18, 0xb7, 0xfd, 0x62, 0x1e,
			0xb7, 0xe5, 0x63, 0x38, 0xc5, 0xfb, 0xff, 0x71, 0xd9,
			0x1d, 0x17, 0x22, 0xda, 0x58, 0xf1, 0x0f, 0x9e, 0x8f,
			0x41, 0x2f, 0x39, 0x9c, 0xb3, 0x06, 0x70, 0xa7, 0x27,
			0xe9, 0x91, 0x94, 0xaa, 0x69, 0x27, 0xaf, 0xf2, 0x54,
			0x16, 0xec, 0x48, 0x9d, 0x45, 0x3a, 0x80, 0x7e, 0x03,
			0xc0, 0x83}},
		script: []byte{btcscript.OP_HASH160},
		after: [][]byte{{0x8b, 0xfa, 0x5c, 0x1f, 0x68, 0x5f, 0x13, 0x86, 0x3e,
			0x74, 0x2e, 0x1b, 0xaf, 0x15, 0xf1, 0x71, 0xad, 0x49,
			0x8b, 0x8f}},
		disassembly: "OP_HASH160",
	},
	// now with pushing. (mostly to check the disassembly)
	{
		name:   "op_hash160 full script",
		before: [][]byte{},
		script: []byte{btcscript.OP_DATA_65,
			0x04, 0x0f, 0xa4, 0x92, 0xe3, 0x59, 0xde, 0xe8, 0x4b,
			0x53, 0xfe, 0xc5, 0xe9, 0x18, 0xb7, 0xfd, 0x62, 0x1e,
			0xb7, 0xe5, 0x63, 0x38, 0xc5, 0xfb, 0xff, 0x71, 0xd9,
			0x1d, 0x17, 0x22, 0xda, 0x58, 0xf1, 0x0f, 0x9e, 0x8f,
			0x41, 0x2f, 0x39, 0x9c, 0xb3, 0x06, 0x70, 0xa7, 0x27,
			0xe9, 0x91, 0x94, 0xaa, 0x69, 0x27, 0xaf, 0xf2, 0x54,
			0x16, 0xec, 0x48, 0x9d, 0x45, 0x3a, 0x80, 0x7e, 0x03,
			0xc0, 0x83,
			btcscript.OP_HASH160, btcscript.OP_DATA_20,
			0x8b, 0xfa, 0x5c, 0x1f, 0x68, 0x5f, 0x13, 0x86, 0x3e,
			0x74, 0x2e, 0x1b, 0xaf, 0x15, 0xf1, 0x71, 0xad, 0x49,
			0x8b, 0x8f,
			btcscript.OP_EQUALVERIFY},
		after:       [][]byte{},
		disassembly: "040fa492e359dee84b53fec5e918b7fd621eb7e56338c5fbff71d91d1722da58f10f9e8f412f399cb30670a727e99194aa6927aff25416ec489d453a807e03c083 OP_HASH160 8bfa5c1f685f13863e742e1baf15f171ad498b8f OP_EQUALVERIFY",
	},
	{
		name:           "op_hash160 no args",
		script:         []byte{btcscript.OP_HASH160},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_HASH160",
	},
	// hash256 test taken from spend of:
	// 09f691b2263260e71f363d1db51ff3100d285956a40cc0e4f8c8c2c4a80559b1
	{
		name: "op_hash256",
		before: [][]byte{{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
			0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61, 0x7f,
			0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f,
			0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a, 0x29, 0xab, 0x5f,
			0x49, 0xff, 0xff, 0x00, 0x1d, 0x1d, 0xac, 0x2b, 0x7c}},
		script: []byte{btcscript.OP_HASH256},
		after: [][]byte{{0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72, 0xc1,
			0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f, 0x93, 0x1e,
			0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c, 0x68, 0xd6, 0x19,
			0x00, 0x00, 0x00, 0x00, 0x00}},
		disassembly: "OP_HASH256",
	},
	{
		name:           "OP_HASH256 no args",
		script:         []byte{btcscript.OP_HASH256},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_HASH256",
	},
	// We need a more involved setup to test OP_CHECKSIG and
	// OP_CHECKMULTISIG (see script_test.go) but we can test it with
	// invalid arguments here quite easily.
	{
		name:           "OP_CHECKSIG one arg",
		script:         []byte{btcscript.OP_1, btcscript.OP_CHECKSIG},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_1 OP_CHECKSIG",
	},
	{
		name:           "OP_CHECKSIG no arg",
		script:         []byte{btcscript.OP_CHECKSIG},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_CHECKSIG",
	},
	{
		name: "OP_CHECKSIGVERIFY one arg",
		script: []byte{btcscript.OP_1,
			btcscript.OP_CHECKSIGVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_1 OP_CHECKSIGVERIFY",
	},
	{
		name:           "OP_CHECKSIGVERIFY no arg",
		script:         []byte{btcscript.OP_CHECKSIGVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_CHECKSIGVERIFY",
	},
	{
		name:           "OP_CHECK_MULTISIG no args",
		script:         []byte{btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_CHECK_MULTISIG",
	},
	{
		name: "OP_CHECK_MULTISIG huge number",
		script: []byte{btcscript.OP_PUSHDATA1,
			0x9, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9,
			btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrNumberTooBig,
		disassembly:    "010203040506070809 OP_CHECK_MULTISIG",
	},
	{
		name: "OP_CHECK_MULTISIG too many keys",
		script: []byte{btcscript.OP_DATA_1, 21,
			btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrTooManyPubkeys,
		disassembly:    "15 OP_CHECK_MULTISIG",
	},
	{
		name: "OP_CHECK_MULTISIG lying about pubkeys",
		script: []byte{btcscript.OP_1,
			btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_1 OP_CHECK_MULTISIG",
	},
	{
		// pubkey comes from blockchain
		name: "OP_CHECK_MULTISIG no sigs",
		script: []byte{
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECK_MULTISIG",
	},
	{
		// pubkey comes from blockchain
		name: "OP_CHECK_MULTISIG sigs huge no",
		script: []byte{
			btcscript.OP_PUSHDATA1,
			0x9, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9,
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrNumberTooBig,
		disassembly:    "010203040506070809 04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECK_MULTISIG",
	},
	{
		name: "OP_CHECK_MULTISIG too few sigs",
		script: []byte{btcscript.OP_1,
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECK_MULTISIG},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_1 04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECK_MULTISIG",
	},
	{
		// pubkey and sig comes from blockchain, are unrelated
		name: "OP_CHECK_MULTISIG won't verify",
		script: []byte{btcscript.OP_1,
			btcscript.OP_DATA_71,
			0x30, 0x44, 0x02, 0x20, 0x4e, 0x45, 0xe1, 0x69, 0x32,
			0xb8, 0xaf, 0x51, 0x49, 0x61, 0xa1, 0xd3, 0xa1, 0xa2,
			0x5f, 0xdf, 0x3f, 0x4f, 0x77, 0x32, 0xe9, 0xd6, 0x24,
			0xc6, 0xc6, 0x15, 0x48, 0xab, 0x5f, 0xb8, 0xcd, 0x41,
			0x02, 0x20, 0x18, 0x15, 0x22, 0xec, 0x8e, 0xca, 0x07,
			0xde, 0x48, 0x60, 0xa4, 0xac, 0xdd, 0x12, 0x90, 0x9d,
			0x83, 0x1c, 0xc5, 0x6c, 0xbb, 0xac, 0x46, 0x22, 0x08,
			0x22, 0x21, 0xa8, 0x76, 0x8d, 0x1d, 0x09, 0x01,
			btcscript.OP_1,
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECK_MULTISIG},
		after:       [][]byte{{0}},
		disassembly: "OP_1 304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901 OP_1 04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECK_MULTISIG",
	},
	{
		// invalid pubkey means that it fails to validate, not an
		// error.  There are pubkeys in the blockchain that don't
		// parse with any validity.
		name: "OP_CHECK_MULTISIG sigs bad pubkey",
		script: []byte{btcscript.OP_1,
			btcscript.OP_DATA_71,
			0x30, 0x44, 0x02, 0x20, 0x4e, 0x45, 0xe1, 0x69, 0x32,
			0xb8, 0xaf, 0x51, 0x49, 0x61, 0xa1, 0xd3, 0xa1, 0xa2,
			0x5f, 0xdf, 0x3f, 0x4f, 0x77, 0x32, 0xe9, 0xd6, 0x24,
			0xc6, 0xc6, 0x15, 0x48, 0xab, 0x5f, 0xb8, 0xcd, 0x41,
			0x02, 0x20, 0x18, 0x15, 0x22, 0xec, 0x8e, 0xca, 0x07,
			0xde, 0x48, 0x60, 0xa4, 0xac, 0xdd, 0x12, 0x90, 0x9d,
			0x83, 0x1c, 0xc5, 0x6c, 0xbb, 0xac, 0x46, 0x22, 0x08,
			0x22, 0x21, 0xa8, 0x76, 0x8d, 0x1d, 0x09, 0x01,
			btcscript.OP_1,
			btcscript.OP_1, btcscript.OP_1,
			btcscript.OP_CHECK_MULTISIG},
		after:       [][]byte{{0}},
		disassembly: "OP_1 304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901 OP_1 OP_1 OP_1 OP_CHECK_MULTISIG",
	},
	// XXX(oga) Test multisig when extra arg is missing. needs valid sig.
	// disabled opcodes
	{
		name:           "OP_CHECKMULTISIGVERIFY no args",
		script:         []byte{btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_CHECKMULTISIGVERIFY",
	},
	{
		name: "OP_CHECKMULTISIGVERIFY huge number",
		script: []byte{btcscript.OP_PUSHDATA1,
			0x9, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrNumberTooBig,
		disassembly:    "010203040506070809 OP_CHECKMULTISIGVERIFY",
	},
	{
		name: "OP_CHECKMULTISIGVERIFY too many keys",
		script: []byte{btcscript.OP_DATA_1, 21,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrTooManyPubkeys,
		disassembly:    "15 OP_CHECKMULTISIGVERIFY",
	},
	{
		name: "OP_CHECKMULTISIGVERIFY lying about pubkeys",
		script: []byte{btcscript.OP_1,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_1 OP_CHECKMULTISIGVERIFY",
	},
	{
		// pubkey comes from blockchain
		name: "OP_CHECKMULTISIGVERIFY no sigs",
		script: []byte{
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECKMULTISIGVERIFY",
	},
	{
		name: "OP_CHECKMULTISIGVERIFY sigs huge no",
		script: []byte{
			btcscript.OP_PUSHDATA1,
			0x9, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9,
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrNumberTooBig,
		disassembly:    "010203040506070809 04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECKMULTISIGVERIFY",
	},
	{
		name: "OP_CHECKMULTISIGVERIFY too few sigs",
		script: []byte{btcscript.OP_1,
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrUnderflow,
		disassembly:    "OP_1 04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECKMULTISIGVERIFY",
	},
	{
		// pubkey and sig comes from blockchain, are unrelated
		name: "OP_CHECKMULTISIGVERIFY won't verify",
		script: []byte{btcscript.OP_1,
			btcscript.OP_DATA_71,
			0x30, 0x44, 0x02, 0x20, 0x4e, 0x45, 0xe1, 0x69, 0x32,
			0xb8, 0xaf, 0x51, 0x49, 0x61, 0xa1, 0xd3, 0xa1, 0xa2,
			0x5f, 0xdf, 0x3f, 0x4f, 0x77, 0x32, 0xe9, 0xd6, 0x24,
			0xc6, 0xc6, 0x15, 0x48, 0xab, 0x5f, 0xb8, 0xcd, 0x41,
			0x02, 0x20, 0x18, 0x15, 0x22, 0xec, 0x8e, 0xca, 0x07,
			0xde, 0x48, 0x60, 0xa4, 0xac, 0xdd, 0x12, 0x90, 0x9d,
			0x83, 0x1c, 0xc5, 0x6c, 0xbb, 0xac, 0x46, 0x22, 0x08,
			0x22, 0x21, 0xa8, 0x76, 0x8d, 0x1d, 0x09, 0x01,
			btcscript.OP_1,
			btcscript.OP_DATA_65,
			0x04, 0xae, 0x1a, 0x62, 0xfe, 0x09, 0xc5, 0xf5, 0x1b,
			0x13, 0x90, 0x5f, 0x07, 0xf0, 0x6b, 0x99, 0xa2, 0xf7,
			0x15, 0x9b, 0x22, 0x25, 0xf3, 0x74, 0xcd, 0x37, 0x8d,
			0x71, 0x30, 0x2f, 0xa2, 0x84, 0x14, 0xe7, 0xaa, 0xb3,
			0x73, 0x97, 0xf5, 0x54, 0xa7, 0xdf, 0x5f, 0x14, 0x2c,
			0x21, 0xc1, 0xb7, 0x30, 0x3b, 0x8a, 0x06, 0x26, 0xf1,
			0xba, 0xde, 0xd5, 0xc7, 0x2a, 0x70, 0x4f, 0x7e, 0x6c,
			0xd8, 0x4c,
			btcscript.OP_1,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrVerifyFailed,
		disassembly:    "OP_1 304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901 OP_1 04ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84c OP_1 OP_CHECKMULTISIGVERIFY",
	},
	{
		// invalid pubkey means that it fails to validate, not an
		// error.  There are pubkeys in the blockchain that don't
		// parse with any validity.
		name: "OP_CHECKMULTISIGVERIFY sigs bad pubkey",
		script: []byte{btcscript.OP_1,
			btcscript.OP_DATA_71,
			0x30, 0x44, 0x02, 0x20, 0x4e, 0x45, 0xe1, 0x69, 0x32,
			0xb8, 0xaf, 0x51, 0x49, 0x61, 0xa1, 0xd3, 0xa1, 0xa2,
			0x5f, 0xdf, 0x3f, 0x4f, 0x77, 0x32, 0xe9, 0xd6, 0x24,
			0xc6, 0xc6, 0x15, 0x48, 0xab, 0x5f, 0xb8, 0xcd, 0x41,
			0x02, 0x20, 0x18, 0x15, 0x22, 0xec, 0x8e, 0xca, 0x07,
			0xde, 0x48, 0x60, 0xa4, 0xac, 0xdd, 0x12, 0x90, 0x9d,
			0x83, 0x1c, 0xc5, 0x6c, 0xbb, 0xac, 0x46, 0x22, 0x08,
			0x22, 0x21, 0xa8, 0x76, 0x8d, 0x1d, 0x09, 0x01,
			btcscript.OP_1,
			btcscript.OP_1, btcscript.OP_1,
			btcscript.OP_CHECKMULTISIGVERIFY},
		expectedReturn: btcscript.StackErrVerifyFailed,
		disassembly:    "OP_1 304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901 OP_1 OP_1 OP_1 OP_CHECKMULTISIGVERIFY",
	},
	{
		name:           "OP_CAT disabled",
		script:         []byte{btcscript.OP_CAT},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_CAT",
	},
	{
		name:           "OP_SUBSTR disabled",
		script:         []byte{btcscript.OP_SUBSTR},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_SUBSTR",
	},
	{
		name:           "OP_LEFT disabled",
		script:         []byte{btcscript.OP_LEFT},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_LEFT",
	},
	{
		name:           "OP_RIGHT disabled",
		script:         []byte{btcscript.OP_RIGHT},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_RIGHT",
	},
	{
		name:           "OP_INVERT disabled",
		script:         []byte{btcscript.OP_INVERT},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_INVERT",
	},
	{
		name:           "OP_AND disabled",
		script:         []byte{btcscript.OP_AND},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_AND",
	},
	{
		name:           "OP_OR disabled",
		script:         []byte{btcscript.OP_OR},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_OR",
	},
	{
		name:           "OP_XOR disabled",
		script:         []byte{btcscript.OP_XOR},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_XOR",
	},
	{
		name:           "OP_2MUL disabled",
		script:         []byte{btcscript.OP_2MUL},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_2MUL",
	},
	{
		name:           "OP_2DIV disabled",
		script:         []byte{btcscript.OP_2DIV},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_2DIV",
	},
	{
		name:           "OP_2DIV disabled",
		script:         []byte{btcscript.OP_2DIV},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_2DIV",
	},
	{
		name:           "OP_MUL disabled",
		script:         []byte{btcscript.OP_MUL},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_MUL",
	},
	{
		name:           "OP_DIV disabled",
		script:         []byte{btcscript.OP_DIV},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_DIV",
	},
	{
		name:           "OP_MOD disabled",
		script:         []byte{btcscript.OP_MOD},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_MOD",
	},
	{
		name:           "OP_LSHIFT disabled",
		script:         []byte{btcscript.OP_LSHIFT},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_LSHIFT",
	},
	{
		name:           "OP_RSHIFT disabled",
		script:         []byte{btcscript.OP_RSHIFT},
		expectedReturn: btcscript.StackErrOpDisabled,
		disassembly:    "OP_RSHIFT",
	},
	// Reserved opcodes
	{
		name:           "OP_RESERVED reserved",
		script:         []byte{btcscript.OP_RESERVED},
		expectedReturn: btcscript.StackErrReservedOpcode,
		disassembly:    "OP_RESERVED",
	},
	{
		name:           "OP_VER reserved",
		script:         []byte{btcscript.OP_VER},
		expectedReturn: btcscript.StackErrReservedOpcode,
		disassembly:    "OP_VER",
	},
	{
		name:           "OP_VERIF reserved",
		script:         []byte{btcscript.OP_VERIF},
		expectedReturn: btcscript.StackErrReservedOpcode,
		disassembly:    "OP_VERIF",
	},
	{
		name:           "OP_VERNOTIF reserved",
		script:         []byte{btcscript.OP_VERNOTIF},
		expectedReturn: btcscript.StackErrReservedOpcode,
		disassembly:    "OP_VERNOTIF",
	},
	{
		name:           "OP_RESERVED1 reserved",
		script:         []byte{btcscript.OP_RESERVED1},
		expectedReturn: btcscript.StackErrReservedOpcode,
		disassembly:    "OP_RESERVED1",
	},
	{
		name:           "OP_RESERVED2 reserved",
		script:         []byte{btcscript.OP_RESERVED2},
		expectedReturn: btcscript.StackErrReservedOpcode,
		disassembly:    "OP_RESERVED2",
	},
	// Invalid Opcodes
	{
		name:           "invalid opcode 186",
		script:         []byte{186},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 187",
		script:         []byte{187},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 188",
		script:         []byte{188},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 189",
		script:         []byte{189},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 190",
		script:         []byte{190},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 191",
		script:         []byte{191},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 192",
		script:         []byte{192},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 193",
		script:         []byte{193},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 194",
		script:         []byte{194},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 195",
		script:         []byte{195},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 196",
		script:         []byte{196},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 197",
		script:         []byte{197},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 198",
		script:         []byte{198},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 199",
		script:         []byte{199},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 200",
		script:         []byte{200},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 201",
		script:         []byte{201},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 202",
		script:         []byte{202},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 203",
		script:         []byte{203},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 204",
		script:         []byte{204},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 205",
		script:         []byte{205},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 206",
		script:         []byte{206},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 207",
		script:         []byte{207},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 208",
		script:         []byte{208},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 209",
		script:         []byte{209},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 210",
		script:         []byte{210},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 211",
		script:         []byte{211},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 212",
		script:         []byte{212},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 213",
		script:         []byte{213},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 214",
		script:         []byte{214},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 215",
		script:         []byte{215},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 216",
		script:         []byte{216},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 217",
		script:         []byte{217},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 218",
		script:         []byte{218},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 219",
		script:         []byte{219},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 220",
		script:         []byte{220},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 221",
		script:         []byte{221},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 222",
		script:         []byte{222},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 223",
		script:         []byte{223},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 224",
		script:         []byte{224},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 225",
		script:         []byte{225},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 226",
		script:         []byte{226},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 227",
		script:         []byte{227},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 228",
		script:         []byte{228},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 229",
		script:         []byte{229},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 230",
		script:         []byte{230},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 231",
		script:         []byte{231},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 232",
		script:         []byte{232},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 233",
		script:         []byte{233},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 234",
		script:         []byte{234},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 235",
		script:         []byte{235},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 236",
		script:         []byte{236},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 237",
		script:         []byte{237},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 238",
		script:         []byte{238},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 239",
		script:         []byte{239},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 240",
		script:         []byte{240},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 241",
		script:         []byte{241},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 242",
		script:         []byte{242},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 243",
		script:         []byte{243},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 244",
		script:         []byte{244},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 245",
		script:         []byte{245},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 246",
		script:         []byte{246},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 247",
		script:         []byte{247},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 248",
		script:         []byte{248},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 249",
		script:         []byte{249},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 250",
		script:         []byte{250},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 251",
		script:         []byte{251},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode 252",
		script:         []byte{252},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassemblyerr: btcscript.StackErrInvalidOpcode,
	},
	{
		name:           "invalid opcode OP_PUBKEY",
		script:         []byte{btcscript.OP_PUBKEY},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassembly:    "OP_PUBKEY",
	},
	{
		name:           "invalid opcode OP_PUBKEYHASH",
		script:         []byte{btcscript.OP_PUBKEYHASH},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassembly:    "OP_PUBKEYHASH",
	},
	{
		name:           "invalid opcode OP_INVALIDOPCODE",
		script:         []byte{btcscript.OP_INVALIDOPCODE},
		expectedReturn: btcscript.StackErrInvalidOpcode,
		disassembly:    "OP_INVALIDOPCODE",
	},
}

func stacksEqual(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func testOpcode(t *testing.T, test *detailedTest) {
	// mock up fake tx.
	tx := &btcwire.MsgTx{
		Version: 1,
		TxIn: []*btcwire.TxIn{
			&btcwire.TxIn{
				PreviousOutpoint: btcwire.OutPoint{
					Hash:  btcwire.ShaHash{},
					Index: 0xffffffff,
				},
				SignatureScript: []byte{},
				Sequence:        0xffffffff,
			},
		},
		TxOut: []*btcwire.TxOut{
			&btcwire.TxOut{
				Value:    0x12a05f200,
				PkScript: []byte{},
			},
		},
		LockTime: 0,
	}

	tx.TxOut[0].PkScript = test.script

	engine, err := btcscript.NewScript(tx.TxIn[0].SignatureScript,
		tx.TxOut[0].PkScript, 0, tx, 1, false)
	if err != nil {
		if err != test.expectedReturn {
			t.Errorf("Error return not expected %s: %v %v",
				test.name, test.expectedReturn, err)
			return
		}
		return
	}
	engine.SetStack(test.before)
	engine.SetAltStack(test.altbefore)

	// test disassembly engine.
	// pc is at start of script 1, so check that DisasmScript matches
	// DisasmPc. Only run this if we have a disassembly for the test.
	// sine one of them have invalid instruction sequences and won't
	// disassemble.
	var disScript, disPC string
	if test.disassembly != "" {
		var err error
		disScript, err = engine.DisasmScript(1)
		if err != nil {
			t.Errorf("failed to disassemble script for %s: %v",
				test.name, err)
		}
	}

	done := false
	for !done {
		if test.disassembly != "" {
			disCurPC, err := engine.DisasmPC()
			if err != nil {
				t.Errorf("failed to disassemble pc for %s: %v",
					test.name, err)
			}
			disPC += disCurPC + "\n"
		}

		done, err = engine.Step()
		if err != nil {
			if err != test.expectedReturn {
				t.Errorf("Error return not expected %s: %v %v",
					test.name, test.expectedReturn, err)
				return
			}
			return
		}
	}
	if err != test.expectedReturn {
		t.Errorf("Error return not expected %s: %v %v",
			test.name, test.expectedReturn, err)
	}

	if test.disassembly != "" {
		if disScript != disPC {
			t.Errorf("script disassembly doesn't match pc "+
				"disassembly for %s: pc: \"%s\" script: \"%s\"",
				test.name, disScript, disPC)
		}
	}

	after := engine.GetStack()
	if !stacksEqual(after, test.after) {
		t.Errorf("Stacks not equal after %s:\ngot: %v\n exp: %v",
			test.name, spew.Sdump(after), spew.Sdump(test.after))
	}
	altafter := engine.GetAltStack()
	if !stacksEqual(altafter, test.altafter) {
		t.Errorf("AltStacks not equal after %s:\n got: %v\nexp: %v",
			test.name, spew.Sdump(altafter),
			spew.Sdump(test.altafter))
	}
}

func TestOpcodes(t *testing.T) {
	for i := range detailedTests {
		testOpcode(t, &detailedTests[i])
	}
}

func testDisasmString(t *testing.T, test *detailedTest) {
	// mock up fake tx.
	dis, err := btcscript.DisasmString(test.script)
	if err != nil {
		if err != test.disassemblyerr {
			t.Errorf("%s: disassembly got error %v expected %v", test.name,
				err, test.disassemblyerr)
		}
		return
	}
	if test.disassemblyerr != nil {
		t.Errorf("%s: expected error %v, got %s", test.name,
			test.disassemblyerr, dis)
		return
	}
	if dis != test.disassembly {
		t.Errorf("Disassembly for %s doesn't match expected "+
			"got: \"%s\" expected: \"%s\"", test.name, dis,
			test.disassembly)
	}
}

func TestDisasmStrings(t *testing.T) {
	for i := range detailedTests {
		testDisasmString(t, &detailedTests[i])
	}
}
