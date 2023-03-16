package golru

import (
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
		want *node[string, T]
	}
	tests := []testCase[string]{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newNode("", tt.args.val, 0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromHead(t *testing.T) {
	length := 5
	head, _, _ := createChainByNode(length)
	targetList := make([]int, length)
	for i := 0; i < length; i++ {
		targetList[i] = i
	}

	if !reflect.DeepEqual(head.ReadForward(nil), targetList) {
		t.Errorf("expect linked list should be %v, but got %v", targetList, head.ReadForward(nil))
	}
}

func TestFromTail(t *testing.T) {
	length := 5
	_, tail, _ := createChainByNode(length)
	targetList := make([]int, length)
	for i := 0; i < length; i++ {
		targetList[length-1-i] = i
	}

	if !reflect.DeepEqual(tail.ReadBackward(nil), targetList) {
		t.Errorf("expect linked list should be %v, but got %v", targetList, tail.ReadBackward(nil))
	}
}

func TestNode_CutOff(t *testing.T) {
	length := 9
	cutIndex := 4

	head, tail, list := createChainByNode(length)
	list[cutIndex].CutOff()

	forwardList := make([]int, 0)
	backwardList := make([]int, 0)

	for i := 0; i < length; i++ {
		if i == cutIndex {
			continue
		}

		forwardList = append(forwardList, i)
		backwardList = append(backwardList, length-i-1)
	}

	if !reflect.DeepEqual(head.ReadForward(nil), forwardList) {
		t.Errorf("expect forward linked list should be %v, but got %v", forwardList, head.ReadForward(nil))
	}

	if !reflect.DeepEqual(tail.ReadBackward(nil), backwardList) {
		t.Errorf("expect backward linked list should be %v, but got %v", backwardList, tail.ReadBackward(nil))
	}
}

func TestNode_CutOff_Concurrent(t *testing.T) {
	length := 9

	head, tail, list := createChainByNode(length)
	lock := new(sync.Mutex)

	var wg sync.WaitGroup
	for i := 0; i < length; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lock.Lock()
			list[i].CutOff()
			lock.Unlock()
		}(i)
	}
	wg.Wait()

	if !reflect.DeepEqual(head.ReadForward(nil), []int{0}) {
		t.Errorf("expect forward linked list should be %v, but got %v", []int{0}, head.ReadForward(nil))
	}

	if !reflect.DeepEqual(tail.ReadBackward(nil), []int{length - 1}) {
		t.Errorf("expect backward linked list should be %v, but got %v", []int{length - 1}, tail.ReadBackward(nil))
	}
}

func createChainByNode(length int) (head, tail *node[string, int], list []*node[string, int]) {
	if length == 0 {
		return nil, nil, nil
	}

	list = make([]*node[string, int], length)
	for i := 0; i < length; i++ {
		list[i] = newNode("", i, 0)
		if i > 0 && i < length {
			list[i-1].SetNext(list[i])
			list[i].SetPre(list[i-1])
		}
	}
	return list[0], list[length-1], list
}

func TestDoubleLinkedList_Append(t *testing.T) {
	list := newDoubleLinkedList[string, int]()
	list.Append(newNode("", 1, 0))
	list.Prepend(newNode("", 11, 0))
	list.Append(newNode("", 2, 0))
	list.Prepend(newNode("", 22, 0))
	list.Append(newNode("", 3, 0))

	targetList := []int{22, 11, 1, 2, 3}
	headNode, _ := list.Head()
	if got := headNode.ReadForward(list.dummyTail); !reflect.DeepEqual(got, targetList) {
		t.Errorf("expect forward linked list should be %v, but got %v", targetList, got)
	}
}

func TestDoubleLinkedList_Pop(t *testing.T) {
	length := 10
	list := newDoubleLinkedList[string, int]()
	for i := 0; i < length; i++ {
		list.Append(newNode("", i, 0))
	}

	n, ok := list.Pop()
	curVal := length - 1
	for ok {
		if n.Val() != curVal {
			t.Errorf("expect node value should be %v, but got %v", curVal, n.Val())
			return
		}
		n, ok = list.Pop()
		curVal--
	}
}
