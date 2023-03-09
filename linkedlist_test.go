package golru

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func Test_newNode(t *testing.T) {
	type args[T any] struct {
		val T
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want *node[T]
	}
	tests := []testCase[string]{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newNode(tt.args.val, 0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromHead(t *testing.T) {
	head, _, _ := createChainByNode(5)
	fmt.Println(head.ReadForward(nil))
}

func TestFromTail(t *testing.T) {
	_, tail, _ := createChainByNode(5)
	fmt.Println(tail.ReadBackward(nil))
}

func TestNode_CutOff(t *testing.T) {
	head, tail, list := createChainByNode(9)
	list[4].CutOff()

	fmt.Println(head.ReadForward(nil))
	fmt.Println(tail.ReadBackward(nil))
}

func TestNode_CutOff_Concurrent(t *testing.T) {
	head, tail, list := createChainByNode(9)

	var wg sync.WaitGroup
	for i := 0; i < 9; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			list[i].CutOff()
		}(i)
	}
	wg.Wait()

	fmt.Println(head.ReadForward(nil))
	fmt.Println(tail.ReadBackward(nil))
}

func createChainByNode(length int) (head, tail *node[int], list []*node[int]) {
	if length == 0 {
		return nil, nil, nil
	}

	list = make([]*node[int], length)
	for i := 0; i < length; i++ {
		list[i] = newNode(i, 0)
		if i > 0 && i < length {
			list[i-1].SetNext(list[i])
			list[i].SetPre(list[i-1])
		}
	}
	return list[0], list[length-1], list
}

func TestDoubleLinkedList_Append(t *testing.T) {
	list := newDoubleLinkedList[int]()
	list.Append(newNode(1, 0))
	list.Prepend(newNode(11, 0))
	list.Append(newNode(2, 0))
	list.Prepend(newNode(22, 0))
	list.Append(newNode(3, 0))
	fmt.Println(list.Head().ReadForward(list.dummyTail))
}
