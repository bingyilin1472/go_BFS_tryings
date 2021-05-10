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
	Source string `json:"source"`
	Target string `json:"target"`
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

// 佔位用途(不會耗費空間)
type void struct{}

// NodeSet : 這邊因為我想利用map的索引，確認node存在與否，因此我NodeSet才採用map，所以內容是放空的void{}(聽說他不占空間)
var NodeSet = make(map[string]void)
var EdgeList []Edge
var Style []NodeStyle

func GetRelationNodes(relation []string) {

}
func GetNodeStyle(styles map[string]NodeStyle) {
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
	//jsonStr := `{
	//	"node1":{"relationU":[], "relationD":["node2", "node3"]},
	//	"node2":{"relationU":["node1"], "relationD":["node4"]},
	//	"node3":{"relationU":["node1"], "relationD":["node4"]},
	//	"node4":{"relationU":["node2","node3"], "relationD":["node5", "node6"]},
	//	"node5":{"relationU":["node4"], "relationD":["node7"]},
	//	"node6":{"relationU":["node4"], "relationD":["node7"]},
	//	"node7":{"relationU":["node5","node6"], "relationD":[]}
	//}`
	jsonGraph := `{
  "2020-C1_Lu-Ming Rice_Tainan 16": {
    "relationU": [],
    "relationD": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tainan 16"
    ],
    "relationD": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Site Preparation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Delivering"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tainan 16"
    ],
    "relationD": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Site Preparation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Delivering"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tainan 16"
    ],
    "relationD": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Site Preparation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Delivering"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tainan 16"
    ],
    "relationD": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Site Preparation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Delivering"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Site Preparation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Cheng-Po Hsu_Organic Farm","Tractor"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Rice Seed Transplantation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Cheng-Po Hsu_Organic Farm","Rice Transplanter"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Cheng-Po Hsu_Organic Farm",
      "Drainage",
      "Irrigating"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Cheng-Po Hsu_Organic Farm",
      "Fwusow organic fertilizer 426",
      "Fwusow special fertilizer for organic cultivation 522",
      "Fagopyrum Esculentum Seed",
      "Ear fertilizer"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Biotic Suppression Operation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Cheng-Po Hsu_Organic Farm",
      "Tea Seed Meal"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Rice House",
      "Low Temperature Bin",
      "Low Temperature Paddy Dryer",
      "Brown Rice Milling And Package",
      "Milled Rice And Package",
      "Paddy complete harvester"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Pesticide And Detecting": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Rice House",
      "Pass"
    ]
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Delivering": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu"
    ],
    "relationD": [
      "Rice House"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Site Preparation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Tian-Shang Chang_Organic Farm",
      "Tractor"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Rice Seed Transplantation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Tian-Shang Chang_Organic Farm",
      "Rice Transplanter"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Tian-Shang Chang_Organic Farm",
      "Drainage",
      "Irrigating"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Tian-Shang Chang_Organic Farm",
      "Fwusow organic fertilizer 426",
      "Fwusow special fertilizer for organic cultivation 522",
      "Fagopyrum Esculentum Seed",
      "Ear fertilizer"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Biotic Suppression Operation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Tian-Shang Chang_Organic Farm",
      "Tea Seed Meal"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Rice House",
      "Low Temperature Bin",
      "Low Temperature Paddy Dryer",
      "Brown Rice Milling And Package",
      "Milled Rice And Package",
      "Paddy complete harvester"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Pesticide And Detecting": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Rice House",
      "Pass"
    ]
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Delivering": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang"
    ],
    "relationD": [
      "Rice House"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Site Preparation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Jin-Shi Hsieh_Organic Farm",
      "Tractor"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Rice Seed Transplantation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Jin-Shi Hsieh_Organic Farm",
      "Rice Transplanter"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Jin-Shi Hsieh_Organic Farm",
      "Drainage",
      "Irrigating"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Jin-Shi Hsieh_Organic Farm",
      "Fwusow organic fertilizer 426",
      "Fwusow special fertilizer for organic cultivation 522",
      "Fagopyrum Esculentum Seed",
      "Ear fertilizer"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Biotic Suppression Operation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Jin-Shi Hsieh_Organic Farm",
      "Tea Seed Meal"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Rice House",
      "Low Temperature Bin",
      "Low Temperature Paddy Dryer",
      "Brown Rice Milling And Package",
      "Milled Rice And Package",
      "Paddy complete harvester"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Pesticide And Detecting": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Rice House",
      "Pass"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Delivering": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh"
    ],
    "relationD": [
      "Rice House"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Site Preparation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Jin-Nan Hsieh_Organic Farm",
      "Tractor"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Rice Seed Transplantation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Jin-Nan Hsieh_Organic Farm",
      "Rice Transplanter"

    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Jin-Nan Hsieh_Organic Farm",
      "Drainage",
      "Irrigating"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Jin-Nan Hsieh_Organic Farm",
      "Fwusow organic fertilizer 426",
      "Fwusow special fertilizer for organic cultivation 522",
      "Fagopyrum Esculentum Seed",
      "Ear fertilizer"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Biotic Suppression Operation": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Jin-Nan Hsieh_Organic Farm",
      "Tea Seed Meal"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Rice House",
      "Low Temperature Bin",
      "Low Temperature Paddy Dryer",
      "Brown Rice Milling And Package",
      "Milled Rice And Package",
      "Paddy complete harvester"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Pesticide And Detecting": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Rice House",
      "Pass"
    ]
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Delivering": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh"
    ],
    "relationD": [
      "Rice House"
    ]
  },
  "Rice House": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Delivering",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Delivering",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Delivering",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Delivering"
    ],
    "relationD": []
  },
  "Cheng-Po Hsu_Organic Farm": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Site Preparation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Biotic Suppression Operation"
    ],
    "relationD": []
  },
  "Tian-Shang Chang_Organic Farm": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Site Preparation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Biotic Suppression Operation"
    ],
    "relationD": []
  },
  "Jin-Shi Hsieh_Organic Farm": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Site Preparation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Biotic Suppression Operation"
    ],
    "relationD": []
  },
  "Jin-Nan Hsieh_Organic Farm": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Site Preparation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Biotic Suppression Operation"
    ],
    "relationD": []
  },
  "Tractor": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Site Preparation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Site Preparation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Site Preparation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Site Preparation"
    ],
    "relationD": []
  },
  "Rice Transplanter": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Rice Seed Transplantation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Rice Seed Transplantation"
    ],
    "relationD": []
  },
  "Drainage": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation"
    ],
    "relationD": []
  },
  "Irrigating": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation"
    ],
    "relationD": []
  },
  "Fwusow organic fertilizer 426": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization"
    ],
    "relationD": []
  },
  "Fwusow special fertilizer for organic cultivation 522": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization"
    ],
    "relationD": []
  },
  "Fagopyrum Esculentum Seed": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization"
    ],
    "relationD": []
  },
  "Ear fertilizer": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization"
    ],
    "relationD": []
  },
  "Tea Seed Meal": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Biotic Suppression Operation",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Biotic Suppression Operation"
    ],
    "relationD": []
  },
  "Low Temperature Bin": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling"
    ],
    "relationD": []
  },
  "Low Temperature Paddy Dryer": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling"
    ],
    "relationD": []
  },
  "Brown Rice Milling And Package": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling"
    ],
    "relationD": []
  },
  "Milled Rice And Package": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling"
    ],
    "relationD": []
  },
  "Paddy complete harvester": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling"
    ],
    "relationD": []
  },
  "Pass": {
    "relationU": [
      "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Pesticide And Detecting",
      "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Pesticide And Detecting"
    ],
    "relationD": []
  }
}`
	jsonStyle := `{
  "2020-C1_Lu-Ming Rice_Tainan 16": {
    "id": "2020-C1_Lu-Ming Rice_Tainan 16",
    "groupId": "product",
    "size": 170,
    "label": "2020-C1\nLu-Ming Rice\nTainan 16",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#413960"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu",
    "groupId": "product",
    "size": 170,
    "label": "2020-C1\nLu-Ming Rice\nCheng-Po Hsu",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#fe917d"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang",
    "groupId": "product",
    "size": 170,
    "label": "2020-C1\nLu-Ming Rice\nTian-Shang Chang",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#fe917d"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh",
    "groupId": "product",
    "size": 170,
    "label": "2020-C1\nLu-Ming Rice\nJin-Shi Hsieh",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#fe917d"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh",
    "groupId": "product",
    "size": 170,
    "label": "2020-C1\nLu-Ming Rice\nJin-Nan Hsieh",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#fe917d"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Site Preparation": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Site Preparation",
    "groupId": "operation",
    "size": 170,
    "label": "Site Preparation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Rice Seed Transplantation": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Rice Seed Transplantation",
    "groupId": "operation",
    "size": 170,
    "label": "Rice Seed\n Transplantation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Irrigation",
    "groupId": "operation",
    "size": 170,
    "label": "Irrigation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Fertilization",
    "groupId": "operation",
    "size": 170,
    "label": "Fertilization",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Biotic Suppression Operation": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Biotic Suppression Operation",
    "groupId": "operation",
    "size": 170,
    "label": "Biotic\n Suppression Operation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Harvesting And Milling",
    "groupId": "operation",
    "size": 170,
    "label": "Harvesting\n And Milling",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Pesticide And Detecting": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Pesticide And Detecting",
    "groupId": "operation",
    "size": 170,
    "label": "Pesticide\n And Detecting",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Delivering": {
    "id": "2020-C1_Lu-Ming Rice_Cheng-Po Hsu_Delivering",
    "groupId": "operation",
    "size": 170,
    "label": "Delivering",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Site Preparation": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Site Preparation",
    "groupId": "operation",
    "size": 170,
    "label": "Site Preparation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Rice Seed Transplantation": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Rice Seed Transplantation",
    "groupId": "operation",
    "size": 170,
    "label": "Rice Seed\n Transplantation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Irrigation",
    "groupId": "operation",
    "size": 170,
    "label": "Irrigation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Fertilization",
    "groupId": "operation",
    "size": 170,
    "label": "Fertilization",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Biotic Suppression Operation": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Biotic Suppression Operation",
    "groupId": "operation",
    "size": 170,
    "label": "Biotic\n Suppression Operation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Harvesting And Milling",
    "groupId": "operation",
    "size": 170,
    "label": "Harvesting\n And Milling",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Pesticide And Detecting": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Pesticide And Detecting",
    "groupId": "operation",
    "size": 170,
    "label": "Pesticide\n And Detecting",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Delivering": {
    "id": "2020-C1_Lu-Ming Rice_Tian-Shang Chang_Delivering",
    "groupId": "operation",
    "size": 170,
    "label": "Delivering",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Site Preparation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Site Preparation",
    "groupId": "operation",
    "size": 170,
    "label": "Site Preparation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Rice Seed Transplantation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Rice Seed Transplantation",
    "groupId": "operation",
    "size": 170,
    "label": "Rice Seed\n Transplantation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Irrigation",
    "groupId": "operation",
    "size": 170,
    "label": "Irrigation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Fertilization",
    "groupId": "operation",
    "size": 170,
    "label": "Fertilization",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Biotic Suppression Operation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Biotic Suppression Operation",
    "groupId": "operation",
    "size": 170,
    "label": "Biotic\n Suppression Operation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Harvesting And Milling",
    "groupId": "operation",
    "size": 170,
    "label": "Harvesting\n And Milling",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Pesticide And Detecting": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Pesticide And Detecting",
    "groupId": "operation",
    "size": 170,
    "label": "Pesticide\n And Detecting",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Delivering": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Shi Hsieh_Delivering",
    "groupId": "operation",
    "size": 170,
    "label": "Delivering",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Site Preparation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Site Preparation",
    "groupId": "operation",
    "size": 170,
    "label": "Site Preparation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Rice Seed Transplantation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Rice Seed Transplantation",
    "groupId": "operation",
    "size": 170,
    "label": "Rice Seed\n Transplantation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Irrigation",
    "groupId": "operation",
    "size": 170,
    "label": "Irrigation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Fertilization",
    "groupId": "operation",
    "size": 170,
    "label": "Fertilization",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Biotic Suppression Operation": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Biotic Suppression Operation",
    "groupId": "operation",
    "size": 170,
    "label": "Biotic\n Suppression Operation",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Harvesting And Milling",
    "groupId": "operation",
    "size": 170,
    "label": "Harvesting\n And Milling",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Pesticide And Detecting": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Pesticide And Detecting",
    "groupId": "operation",
    "size": 170,
    "label": "Pesticide\n And Detecting",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Delivering": {
    "id": "2020-C1_Lu-Ming Rice_Jin-Nan Hsieh_Delivering",
    "groupId": "operation",
    "size": 170,
    "label": "Delivering",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#85d0a0"
    }
  },
  "Rice House": {
    "id": "Rice House",
    "groupId": "farm",
    "size": 170,
    "label": "Rice House",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#ffffff"
    }
  },
  "Cheng-Po Hsu_Organic Farm": {
    "id": "Cheng-Po Hsu_Organic Farm",
    "groupId": "farm",
    "size": 170,
    "label": "Cheng-Po Hsu\nOrganic Farm",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#ffffff"
    }
  },
  "Tian-Shang Chang_Organic Farm": {
    "id": "Tian-Shang Chang_Organic Farm",
    "groupId": "farm",
    "size": 170,
    "label": "Tian-Shang Chang\nOrganic Farm",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#ffffff"
    }
  },
  "Jin-Shi Hsieh_Organic Farm": {
    "id": "Jin-Shi Hsieh_Organic Farm",
    "groupId": "farm",
    "size": 170,
    "label": "Jin-Shi Hsieh\nOrganic Farm",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#ffffff"
    }
  },
  "Jin-Nan Hsieh_Organic Farm": {
    "id": "Jin-Nan Hsieh_Organic Farm",
    "groupId": "farm",
    "size": 170,
    "label": "Jin-Nan Hsieh\nOrganic Farm",
    "labelCfg": {
      "style": {
        "fontSize": 16
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#ffffff"
    }
  },
  "Tractor": {
    "id": "Tractor",
    "groupId": "detail",
    "size": 170,
    "label": "Tractor",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Rice Transplanter": {
    "id": "Rice Transplanter",
    "groupId": "detail",
    "size": 170,
    "label": "Rice \nTransplanter",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Drainage": {
    "id": "Drainage",
    "groupId": "detail",
    "size": 170,
    "label": "Drainage",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Irrigating": {
    "id": "Irrigating",
    "groupId": "detail",
    "size": 170,
    "label": "Irrigating",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Fwusow organic fertilizer 426": {
    "id": "Fwusow organic fertilizer 426",
    "groupId": "detail",
    "size": 170,
    "label": "Fwusow organic\n fertilizer 426",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Fwusow special fertilizer for organic cultivation 522": {
    "id": "Fwusow special fertilizer for organic cultivation 522",
    "groupId": "detail",
    "size": 170,
    "label": "Fwusow special\n fertilizer for\n organic cultivation\n 522",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Fagopyrum Esculentum Seed": {
    "id": "Fagopyrum Esculentum Seed",
    "groupId": "detail",
    "size": 170,
    "label": "Fagopyrum \nEsculentum Seed",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Ear fertilizer": {
    "id": "Ear fertilizer",
    "groupId": "detail",
    "size": 170,
    "label": "Ear fertilizer",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Tea Seed Meal": {
    "id": "Tea Seed Meal",
    "groupId": "detail",
    "size": 170,
    "label": "Tea Seed Meal",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Low Temperature Bin": {
    "id": "Low Temperature Bin",
    "groupId": "detail",
    "size": 170,
    "label": "Low Temperature\n Bin",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Low Temperature Paddy Dryer": {
    "id": "Low Temperature Paddy Dryer",
    "groupId": "detail",
    "size": 170,
    "label": "Low Temperature\n Paddy Dryer",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Brown Rice Milling And Package": {
    "id": "Brown Rice Milling And Package",
    "groupId": "detail",
    "size": 170,
    "label": "Brown Rice \nMilling And\n Package",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Milled Rice And Package": {
    "id": "Milled Rice And Package",
    "groupId": "detail",
    "size": 170,
    "label": "Milled Rice \nAnd Package",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Paddy complete harvester": {
    "id": "Paddy complete harvester",
    "groupId": "detail",
    "size": 170,
    "label": "Paddy complete\n harvester",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  },
  "Pass": {
    "id": "Pass",
    "groupId": "detail",
    "size": 170,
    "label": "Pass",
    "labelCfg": {
      "style": {
        "fontSize": 16,
        "fill": "#ffffff"
      }
    },
    "style": {
      "stroke": "#413960",
      "fill": "#6154a5"
    }
  }
}`

	errGraph := json.Unmarshal([]byte(jsonGraph), &maps)
	if errGraph != nil {
		log.Println(fmt.Errorf("failed to Unmarshal json graph : %s", errGraph.Error()))
	}
	errStyle := json.Unmarshal([]byte(jsonStyle), &styles)
	if errStyle != nil {
		log.Println(fmt.Errorf("failed to Unmarshal json style : %s", errStyle.Error()))
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

	UpMove("2020-C1_Lu-Ming Rice_Cheng-Po Hsu", maps, 2)
	DownMove("2020-C1_Lu-Ming Rice_Cheng-Po Hsu", maps, 10)
	GetNodeStyle(styles)
	log.Println("This NodeSet: ", NodeSet)
	log.Println("This StyleSet", Style)

	tjsonStyle, tsError := json.Marshal(Style)
	if tsError != nil {
		log.Println("style transformed error: ", tsError.Error())
	} else {
		log.Println("style json transformed: ", string(tjsonStyle))
	}

	//log.Println(EdgeList)
	//// range即可窮盡取出nodeKey，不使用value，_
	//for k, _ := range NodeSet {
	//	log.Print(k)
	//}

	// golang中空結構是不佔用記憶體
	//type void struct{}
	//set := make(map[string]void)
	//set["a1"] = void{}
	//if _, ok := set["a2"]; !ok {
	//	log.Println("The key is not existed")
	//}
	//if _, ok := set["a1"]; ok {
	//	log.Println("The key is existed")
	//}

}
