package main

import (
	"encoding/json"
	"fmt"
	"log"
)

// 資料格式規劃: 如以下
// 因為若節點都全名會變很長，譬如由id為 "2021C2_人_產品_操作名稱"，太長
// 直接把他取一個label名稱，譬如只留下操作名稱
// 或者像是2021C2_人_產品的label為人 \n 農產品名稱

// 為了讓判斷是否所有操作完成
// fields: 產品名稱、jsonArray of all types of operations

// 會有一個節點的總圖table，multi-fields
// nodeID、label、type(tableName)
// 不同type的node會有該
// 目前設計node詳細資料是單筆查，點擊節點才會觸發
// 沒有大量掃描的情況，切換table查指定nodeID不至於太難
// (若要查某產品下節點，只設置下level，上則為0，0就不觸發if條件上爬)

// {node ID, relationU{} relationD{} , ....}
// 會有一個serial 作為圖新舊，可以用取最大serial
// SELECT my_id, col2, col3
// FROM mytable
// order by my_id desc
// limit 1

// 總圖樣式table
// nodeID、jsonb of nodeID's style config

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
	// 建立一個0大小的list，item包含Node str/From str/Distance int
	queue := make([]Item, 0)
	// 移動距離初始化為0，置到累積到level要求距離
	var d int64
	d = 0
	// 初始將出發節點nodeName放進來，From root表示非來自他人
	queue = append(queue, Item{Node: nodeName, From: "root", Distance: d})
	top := queue[0]
	// 下面是確認NodeSet沒有該node
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
		// 這是取出最上面的，然後以位址指派方式，模擬一個queue取出的模式
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

	DownMove("node4", maps, 10)
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
