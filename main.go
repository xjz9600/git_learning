package main

import "fmt"

type handler func(i interface{})

type middleware func(handler) handler

func main() {
	var mids []middleware
	mids = append(mids, mid1, mid2())
	a := func(i interface{}) {
		fmt.Println("hello", i)
	}
	for _, mid := range mids {
		a = mid(a)
	}
	a("hello world")
}

func mid1(h handler) handler {
	return func(i interface{}) {
		fmt.Println("first middleware")
		h("word1")
	}
}

func mid2() middleware {
	return func(h handler) handler {
		return func(i interface{}) {
			fmt.Println("second middleware")
			h("word2")
		}
	}
}

type name interface {
}
