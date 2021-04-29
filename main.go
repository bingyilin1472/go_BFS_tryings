package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type NodeRelation struct {
	// json一般都是小寫開頭，可駝峰
	RelationU []string `json:"relationU"`
	RelationD []string `json:"relationD"`
}
type Item struct {
	Node     string
	From     string
	Distance int64
}
type Edge struct {
	Source string
	Target string
}

//queue := make([]int, 0)
//// Push to the queue
//queue = append(queue, 1)
//// Top (just get next element, don't remove it)
//x = queue[0]
//// Discard top element
//queue = queue[1:]
//// Is empty ?
//if len(queue) == 0 {
//fmt.Println("Queue is empty !")
//}

//{
//	"node1":{"relationU":[], "relationD":["node2", "node3"]}
//	"node2":{"relationU":["node1"], "relationD":["node4"]},
//	"node3":{"relationU":["node1"], "relationD":["node4"]},
//	"node4":{"relationU":["node2","node3"], "relationD":["node5", "node6"]},
//	"node5":{"relationU":["node4"], "relationD":["node7"]},
//	"node6":{"relationU":["node4"], "relationD":["node7"]},
//	"node7":{"relationU":["node5","node6"], "relationD":[]}
//}

// 會有renderNode，用於最終回傳給前端list
// 會有renderEdge可能會有兩個不同方向(但我只想一個)
// 會有一個渲染的node、edge list不斷render加進去，最後會Marshal成json回傳frontend
// 另外會有一個實際node map struct(包含名稱以及包含換行...)該node的樣時

// 另外移動會有UP跟down兩種，都會queue取出的時候產生edge以及node
// edge會因為UP或down在render edge時候產生反向的target與source

type void struct{}

var NodeSet = make(map[string]void)
var EdgeList []Edge

func GetRelationNodes(relation []string) {

}
func DownMove(nodeName string, maps map[string]NodeRelation, level int64) {
	queue := make([]Item, 0)
	var d int64
	d = 0
	queue = append(queue, Item{Node: nodeName, From: "root", Distance: d})
	top := queue[0]
	if _, ok := NodeSet[top.Node]; !ok {
		NodeSet[top.Node] = void{}
	}
	queue = queue[1:]
	if len(maps[top.Node].RelationD) != 0 {
		for _, value := range maps[top.Node].RelationD {
			queue = append(queue, Item{Node: value, From: top.Node, Distance: top.Distance + 1})
		}
	}
	for len(queue) != 0 {
		top = queue[0]
		queue = queue[1:]
		if top.Distance > level {
			break
		}
		EdgeList = append(EdgeList, Edge{Source: top.From, Target: top.Node})
		if _, ok := NodeSet[top.Node]; !ok {
			NodeSet[top.Node] = void{}
		}
		if len(maps[top.Node].RelationD) != 0 {
			for _, value := range maps[top.Node].RelationD {
				queue = append(queue, Item{Node: value, From: top.Node, Distance: top.Distance + 1})
			}
		}
	}

	//queue := make([]int, 0)
	//// Push to the queue
	//queue = append(queue, 1)
	//// Top (just get next element, don't remove it)
	//x = queue[0]
	//// Discard top element
	//queue = queue[1:]
	//// Is empty ?
	//if len(queue) == 0 {
	//	fmt.Println("Queue is empty !")
	//}
}

func main() {
	// {"nodeName" :
	maps := make(map[string]NodeRelation)
	jsonStr := `{
		"node1":{"relationU":[], "relationD":["node2", "node3"]},
		"node2":{"relationU":["node1"], "relationD":["node4"]},
		"node3":{"relationU":["node1"], "relationD":["node4"]},
		"node4":{"relationU":["node2","node3"], "relationD":["node5", "node6"]},
		"node5":{"relationU":["node4"], "relationD":["node7"]},
		"node6":{"relationU":["node4"], "relationD":["node7"]},
		"node7":{"relationU":["node5","node6"], "relationD":[]}
	}`

	err := json.Unmarshal([]byte(jsonStr), &maps)
	if err != nil {
		log.Println(fmt.Errorf("failed to Unmarshal json str: %s", err.Error()))
	}
	log.Println(maps["node1"])
	log.Println(maps["node1"].RelationD)
	for _, v := range maps["node1"].RelationD {
		log.Println(v)
	}
	for k, v := range maps {
		log.Println(k)
		log.Println(v)
	}
	var sliceAppend []int
	sliceAppend = append(sliceAppend, 1, 2, 3)
	log.Println(len(sliceAppend))
	sliceAppend2 := []int{4, 5, 6}
	log.Println(sliceAppend2, len(sliceAppend2))
	// ...可以將切片多個值取出作為element1, element2, element3 ...引數
	sliceAppend = append(sliceAppend, sliceAppend2...)
	log.Println(sliceAppend)

	items := make([]Item, 0)
	items = append(items, Item{Node: "NodeName1", From: "NodeName1", Distance: 1})
	log.Println(items)

	DownMove("node4", maps, 2)
	log.Println(NodeSet)
	log.Println(EdgeList)
	for k, _ := range NodeSet {
		log.Print(k)
	}

	// golang中空結構是不佔用記憶體
	type void struct{}
	set := make(map[string]void)
	set["a1"] = void{}
	if _, ok := set["a2"]; !ok {
		log.Println("The key is not existed")
	}
	if _, ok := set["a1"]; ok {
		log.Println("The key is existed")
	}

}
