// Copyright (c) 2013 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcscript_test

import (
	"bytes"
	"errors"
	"github.com/conformal/btcscript"
	"math/big"
	"testing"
)

type stackTest struct {
	name           string
	before         [][]byte
	operation      func(*btcscript.Stack) error
	expectedReturn error
	after          [][]byte
}

var stackTests = []stackTest{
	{
		"noop",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			return nil
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}, {5}},
	},
	{
		"peek underflow (byte)",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			_, err := stack.PeekByteArray(5)
			return err
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"peek underflow (int)",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			_, err := stack.PeekInt(5)
			return err
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"peek underflow (bool)",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			_, err := stack.PeekBool(5)
			return err
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"pop",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			val, err := stack.PopByteArray()
			if err != nil {
				return err
			}
			if !bytes.Equal(val, []byte{5}) {
				return errors.New("not equal!")
			}
			return err
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}},
	},
	{
		"pop",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			val, err := stack.PopByteArray()
			if err != nil {
				return err
			}
			if !bytes.Equal(val, []byte{5}) {
				return errors.New("not equal!")
			}
			return err
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}},
	},
	{
		"pop everything",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			for i := 0; i < 5; i++ {
				_, err := stack.PopByteArray()
				if err != nil {
					return err
				}
			}
			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"pop underflow",
		[][]byte{{1}, {2}, {3}, {4}, {5}},
		func(stack *btcscript.Stack) error {
			for i := 0; i < 6; i++ {
				_, err := stack.PopByteArray()
				if err != nil {
					return err
				}
			}
			return nil
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"pop bool",
		[][]byte{{0}},
		func(stack *btcscript.Stack) error {
			val, err := stack.PopBool()
			if err != nil {
				return err
			}

			if val != false {
				return errors.New("unexpected value")
			}
			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"pop bool",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			val, err := stack.PopBool()
			if err != nil {
				return err
			}

			if val != true {
				return errors.New("unexpected value")
			}
			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"pop bool",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			_, err := stack.PopBool()
			if err != nil {
				return err
			}

			return nil
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	// XXX test popInt -> byte format matters here.
	{
		"dup",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(1)
			if err != nil {
				return err
			}

			return nil
		},
		nil,
		[][]byte{{1}, {1}},
	},
	{
		"dup2",
		[][]byte{{1}, {2}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(2)
			if err != nil {
				return err
			}

			return nil
		},
		nil,
		[][]byte{{1}, {2}, {1}, {2}},
	},
	{
		"dup3",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(3)
			if err != nil {
				return err
			}

			return nil
		},
		nil,
		[][]byte{{1}, {2}, {3}, {1}, {2}, {3}},
	},
	{
		"dup0",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(0)
			if err != nil {
				return err
			}

			return nil
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"dup-1",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(-1)
			if err != nil {
				return err
			}

			return nil
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"dup too much",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(2)
			if err != nil {
				return err
			}

			return nil
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"dup-1",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			err := stack.DupN(-1)
			if err != nil {
				return err
			}

			return nil
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"PushBool true",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushBool(true)

			return nil
		},
		nil,
		[][]byte{{1}},
	},
	{
		"PushBool false",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushBool(false)

			return nil
		},
		nil,
		[][]byte{{0}},
	},
	{
		"PushBool PopBool",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushBool(true)
			val, err := stack.PopBool()
			if err != nil {
				return err
			}
			if val != true {
				return errors.New("unexpected value")
			}

			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"PushBool PopBool 2",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushBool(false)
			val, err := stack.PopBool()
			if err != nil {
				return err
			}
			if val != false {
				return errors.New("unexpected value")
			}

			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"PushInt PopBool",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushInt(big.NewInt(1))
			val, err := stack.PopBool()
			if err != nil {
				return err
			}
			if val != true {
				return errors.New("unexpected value")
			}

			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"PushInt PopBool 2",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushInt(big.NewInt(0))
			val, err := stack.PopBool()
			if err != nil {
				return err
			}
			if val != false {
				return errors.New("unexpected value")
			}

			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"PushInt PopBool 2",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushInt(big.NewInt(0))
			val, err := stack.PopBool()
			if err != nil {
				return err
			}
			if val != false {
				return errors.New("unexpected value")
			}

			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"Nip top",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.NipN(0)
		},
		nil,
		[][]byte{{1}, {2}},
	},
	{
		"Nip middle",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.NipN(1)
		},
		nil,
		[][]byte{{1}, {3}},
	},
	{
		"Nip low",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.NipN(2)
		},
		nil,
		[][]byte{{2}, {3}},
	},
	{
		"Nip too much",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			// bite off more than we can chew
			return stack.NipN(3)
		},
		btcscript.StackErrUnderflow,
		[][]byte{{2}, {3}},
	},
	{
		"Nip too much",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			// bite off more than we can chew
			return stack.NipN(3)
		},
		btcscript.StackErrUnderflow,
		[][]byte{{2}, {3}},
	},
	{
		"keep on tucking",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.Tuck()
		},
		nil,
		[][]byte{{1}, {3}, {2}, {3}},
	},
	{
		"a little tucked up",
		[][]byte{{1}}, // too few arguments for tuck
		func(stack *btcscript.Stack) error {
			return stack.Tuck()
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"all tucked up",
		[][]byte{}, // too few arguments  for tuck
		func(stack *btcscript.Stack) error {
			return stack.Tuck()
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"drop 1",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.DropN(1)
		},
		nil,
		[][]byte{{1}, {2}, {3}},
	},
	{
		"drop 2",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.DropN(2)
		},
		nil,
		[][]byte{{1}, {2}},
	},
	{
		"drop 3",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.DropN(3)
		},
		nil,
		[][]byte{{1}},
	},
	{
		"drop 4",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.DropN(4)
		},
		nil,
		[][]byte{},
	},
	{
		"drop 4/5",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.DropN(5)
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"drop invalid",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.DropN(0)
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"Rot1",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.RotN(1)
		},
		nil,
		[][]byte{{1}, {3}, {4}, {2}},
	},
	{
		"Rot2",
		[][]byte{{1}, {2}, {3}, {4}, {5}, {6}},
		func(stack *btcscript.Stack) error {
			return stack.RotN(2)
		},
		nil,
		[][]byte{{3}, {4}, {5}, {6}, {1}, {2}},
	},
	{
		"Rot too little",
		[][]byte{{1}, {2}},
		func(stack *btcscript.Stack) error {
			return stack.RotN(1)
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"Rot0",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.RotN(0)
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"Swap1",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.SwapN(1)
		},
		nil,
		[][]byte{{1}, {2}, {4}, {3}},
	},
	{
		"Swap2",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.SwapN(2)
		},
		nil,
		[][]byte{{3}, {4}, {1}, {2}},
	},
	{
		"Swap too little",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			return stack.SwapN(1)
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"Swap0",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.SwapN(0)
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"Over1",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.OverN(1)
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}, {3}},
	},
	{
		"Over2",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.OverN(2)
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}, {1}, {2}},
	},
	{
		"Over too little",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			return stack.OverN(1)
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"Over0",
		[][]byte{{1}, {2}, {3}},
		func(stack *btcscript.Stack) error {
			return stack.OverN(0)
		},
		btcscript.StackErrInvalidArgs,
		[][]byte{},
	},
	{
		"Pick1",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.PickN(1)
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}, {3}},
	},
	{
		"Pick2",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.PickN(2)
		},
		nil,
		[][]byte{{1}, {2}, {3}, {4}, {2}},
	},
	{
		"Pick too little",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			return stack.PickN(1)
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"Roll1",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.RollN(1)
		},
		nil,
		[][]byte{{1}, {2}, {4}, {3}},
	},
	{
		"Roll2",
		[][]byte{{1}, {2}, {3}, {4}},
		func(stack *btcscript.Stack) error {
			return stack.RollN(2)
		},
		nil,
		[][]byte{{1}, {3}, {4}, {2}},
	},
	{
		"Roll too little",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			return stack.RollN(1)
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
	{
		"Peek bool",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			// Peek bool is otherwise pretty well tested, just check
			// it works.
			val, err := stack.PeekBool(0)
			if err != nil {
				return err
			}
			if val != true {
				return errors.New("invalid result")
			}
			return nil
		},
		nil,
		[][]byte{{1}},
	},
	{
		"Peek bool 2",
		[][]byte{{0}},
		func(stack *btcscript.Stack) error {
			// Peek bool is otherwise pretty well tested, just check
			// it works.
			val, err := stack.PeekBool(0)
			if err != nil {
				return err
			}
			if val != false {
				return errors.New("invalid result")
			}
			return nil
		},
		nil,
		[][]byte{{0}},
	},
	{
		"Peek int",
		[][]byte{{1}},
		func(stack *btcscript.Stack) error {
			// Peek int is otherwise pretty well tested, just check
			// it works.
			val, err := stack.PeekInt(0)
			if err != nil {
				return err
			}
			if val.Cmp(big.NewInt(1)) != 0 {
				return errors.New("invalid result")
			}
			return nil
		},
		nil,
		[][]byte{{1}},
	},
	{
		"Peek int 2",
		[][]byte{{0}},
		func(stack *btcscript.Stack) error {
			// Peek int is otherwise pretty well tested, just check
			// it works.
			val, err := stack.PeekInt(0)
			if err != nil {
				return err
			}
			if val.Cmp(big.NewInt(0)) != 0 {
				return errors.New("invalid result")
			}
			return nil
		},
		nil,
		[][]byte{{0}},
	},
	{
		"pop int",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			stack.PushInt(big.NewInt(1))
			// Peek int is otherwise pretty well tested, just check
			// it works.
			val, err := stack.PopInt()
			if err != nil {
				return err
			}
			if val.Cmp(big.NewInt(1)) != 0 {
				return errors.New("invalid result")
			}
			return nil
		},
		nil,
		[][]byte{},
	},
	{
		"pop empty",
		[][]byte{},
		func(stack *btcscript.Stack) error {
			// Peek int is otherwise pretty well tested, just check
			// it works.
			_, err := stack.PopInt()
			return err
		},
		btcscript.StackErrUnderflow,
		[][]byte{},
	},
}

func doTest(t *testing.T, test stackTest) {
	stack := btcscript.Stack{}

	for i := range test.before {
		stack.PushByteArray(test.before[i])
	}
	err := test.operation(&stack)
	if err != test.expectedReturn {
		t.Errorf("%s: operation return not what expected: %v vs %v",
			test.name, err, test.expectedReturn)
	}
	if err != nil {
		return
	}
	if len(test.after) != stack.Depth() {
		t.Errorf("%s: stack depth doesn't match expected: %v vs %v",
			test.name, len(test.after), stack.Depth())
	}
	for i := range test.after {
		val, err := stack.PeekByteArray(stack.Depth() - i - 1)
		if err != nil {
			t.Errorf("%s: can't peek %dth stack entry: %v",
				test.name, i, err)
			break
		}

		if !bytes.Equal(val, test.after[i]) {
			t.Errorf("%s: %dth stack entry doesn't match "+
				"expected: %v vs %v", test.name, i, val,
				test.after[i])
			break
		}
	}
}

func TestStack(t *testing.T) {
	for i := range stackTests {
		doTest(t, stackTests[i])
	}
}
