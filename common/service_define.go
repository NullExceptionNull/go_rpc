package common

type Service struct {
	Name  string  `json:"name"`
	Nodes []*Node `json:"nodes"`
}

type Node struct {
	Name          string `json:"name"` //这里的节点名称和service名称保持一致
	Id            int    `json:"id"`
	Ip            string `json:"ip"`
	Port          int    `json:"port"`
	Weight        int    `json:"weight"`
	LastHeartBeat int64  `json:"last_heart_beat"`
}
