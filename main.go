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

// graph 格式會長成下面這個形式，應該會採用jsonb，因為id順序沒有特別重要，會以key-value做indexing
//{
//	"node1":{"relationU":[], "relationD":["node2", "node3"]}
//	"node2":{"relationU":["node1"], "relationD":["node4"]},
//	"node3":{"relationU":["node1"], "relationD":["node4"]}
//}

type NodeRelation struct {
	// json一般都是小寫開頭，可駝峰
	RelationU []string `json:"relationU"`
	RelationD []string `json:"relationD"`
}

// Item : queue裡面的內容，會有當前node，以及他從哪裡來(為了建構正確的edge方向用途)，以及走多遠
type Item struct {
	Node     string
	From     string
	Distance int64
}
type Edge struct {
	Source string
	Target string
}

type NodeStyle struct {
	Id       string      `json:"id"`
	GroupId  string      `json:"groupId"`
	Size     int64       `json:"size"`
	Label    string      `json:"label"`
	LabelCfg LabelStyle  `json:"labelCfg"`
	Style    CircleStyle `json:"style"`
}
type LabelStyle struct {
	Style FontStyle
}
type FontStyle struct {
	FontSize int64  `json:"fontSize"`
	Fill     string `json:"fill"`
}

type CircleStyle struct {
	Stroke string `json:"stroke"`
	Fill   string `json:"fill"`
}

// list實踐queue的方法
//queue := make([]int, 0) // 透過make創建一個大小為0的list
//// Push to the queue
//queue = append(queue, 1) // 透過append放入queue中，現在放入1，他是逐個遞增append，會得到1|2|3 ...
//// Top (just get next element, don't remove it)
//x = queue[0] // 取出來很單純取0就是最前面的(queue是先進先出，FIFO)，indexing 0即可實踐
//// Discard top element
//queue = queue[1:] // 從第二個做為list address，前面那個放棄掉，這方面的損耗還好(當然code變巨大時候要注意)
//// Is empty ?
//if len(queue) == 0 {  //下面這個可以判斷該list是否為空透過built-in len方法，我們這邊還有distance要素，滿足distance也會結束
//fmt.Println("Queue is empty !")
//}

// 會有renderNode，用於最終回傳給前端list
// 會有renderEdge可能會有兩個不同方向(但我只想一個)
// 會有一個渲染的node、edge list不斷render加進去，最後會Marshal成json回傳frontend
// 另外會有一個實際node map struct(包含名稱以及包含換行...)該node的樣時

// 另外移動會有UP跟down兩種，都會queue取出的時候產生edge以及node
// edge會因為UP或down在render edge時候產生反向的target與source

type void struct{}

// NodeSet : 這邊因為我想利用map的索引，確認node存在與否，因此我NodeSet才採用map，所以內容是放空的void{}(聽說他不占空間)
var NodeSet = make(map[string]void)
var EdgeList []Edge
var Style []NodeStyle

func GetRelationNodes(relation []string) {

}
func GetNodeStyle(styles map[string]NodeStyle) []NodeStyle {
	for k, _ := range NodeSet {
		Style = append(Style, styles[k])
	}
}
func UpMove(nodeName string, maps map[string]NodeRelation, level int64) {
	// 起始點是隨func進來，nodeName
	// 建立一個0大小的list，item包含Node str/From str/Distance int
	queue := make([]Item, 0)
	// 移動距離初始化為0，置到累積到level要求距離
	var d int64
	d = 0
	// 初始將出發節點nodeName放進來，From root表示非來自他人
	queue = append(queue, Item{Node: nodeName, From: "root", Distance: d})
	// 第一次一定會取出，從起始點出發
	top := queue[0]
	// 下面是確認NodeSet沒有該node，透過comma ok，這邊不對NodeSet做事，因此_，因為在判斷存在與否
	if _, ok := NodeSet[top.Node]; !ok {
		NodeSet[top.Node] = void{}
	}
	// queue重整address，再第一次取出後，UpMove只看RelationU
	queue = queue[1:]
	// 看是否能往上，若沒不進到下面移動迴圈(所有演算法都能以forLoop實踐，這是很重要觀念)
	if len(maps[top.Node].RelationU) != 0 {
		// key沒有要用到_，同一個節點取出來的RelationU都是當前距離加1、top.Distance + 1
		// 另一方面也是From: top.Node
		for _, value := range maps[top.Node].RelationU {
			queue = append(queue, Item{Node: value, From: top.Node, Distance: top.Distance + 1})
		}
	}
	// 下面以while方式來持續移動，直到queue為空，裡面也有distance滿足地跳出條件
	for len(queue) != 0 {
		// 這是取出最上面的，然後以位址指派方式，模擬一個queue取出的模式
		top = queue[0]
		queue = queue[1:]
		// 這是先判斷，因為當前queue的node都尚未放入之後要用的nodeSet，若發現他有大於distance的就該跳出，因為不會放入
		if top.Distance > level {
			break
		}
		// UpMove方向是反過來的，From放在Target、Source則是當前Node(top.Node)
		EdgeList = append(EdgeList, Edge{Source: top.Node, Target: top.From})
		// 這邊因為我想利用map的索引，確認node存在與否，因此我NodeSet才採用map，所以內容是放空的void{}(聽說他不占空間)
		if _, ok := NodeSet[top.Node]; !ok {
			// 若不存在就加入該nodeKey:{}
			NodeSet[top.Node] = void{}
		}
		// 檢查該node是否能繼續往上，添入queue
		if len(maps[top.Node].RelationU) != 0 {
			// for窮盡list/array也會有index 0、1、2，不用_
			for _, value := range maps[top.Node].RelationU {
				queue = append(queue, Item{Node: value, From: top.Node, Distance: top.Distance + 1})
			}
		}
	}
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
	// DownMove差異在於往下是看RelationD
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
		// EdgeList，DownMove，source會是From，不用反過來(Up的From則是target)
		EdgeList = append(EdgeList, Edge{Source: top.From, Target: top.Node})
		if _, ok := NodeSet[top.Node]; !ok {
			NodeSet[top.Node] = void{}
		}
		// DownMove差異在於往下是看RelationD
		if len(maps[top.Node].RelationD) != 0 {
			for _, value := range maps[top.Node].RelationD {
				queue = append(queue, Item{Node: value, From: top.Node, Distance: top.Distance + 1})
			}
		}
	}
}

func main() {
	// {"nodeName" :
	maps := make(map[string]NodeRelation)
	styles := make(map[string]NodeStyle)
	jsonStr := `{
		"node1":{"relationU":[], "relationD":["node2", "node3"]},
		"node2":{"relationU":["node1"], "relationD":["node4"]},
		"node3":{"relationU":["node1"], "relationD":["node4"]},
		"node4":{"relationU":["node2","node3"], "relationD":["node5", "node6"]},
		"node5":{"relationU":["node4"], "relationD":["node7"]},
		"node6":{"relationU":["node4"], "relationD":["node7"]},
		"node7":{"relationU":["node5","node6"], "relationD":[]}
	}`
	jsonGraph := ``

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

	UpMove("node4", maps, 2)
	DownMove("node4", maps, 1)
	log.Println("This NodeSet: ", NodeSet)
	log.Println(EdgeList)
	// range即可窮盡取出nodeKey，不使用value，_
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
