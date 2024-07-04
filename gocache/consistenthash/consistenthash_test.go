package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	//定义虚拟节点得倍数，定义哈希函数，自定义的 Hash 算法只处理数字，传入字符串表示的数字，返回对应的数字即可。
	hash := New(3,func(key []byte) uint32 {
		i,_ := strconv.Atoi(string(key))
		return uint32(i)
	})
	// Given the above hash function, this will give replicas with "hashes":
	// 2, 4, 6, 12, 14, 16, 22, 24, 26
	//一开始，有 2/4/6 三个真实节点，对应的虚拟节点的哈希值是 02/12/22、04/14/24、06/16/26。
	hash.Register("6", "4", "2") //添加节点
	
	// 测试用例 2/11/23/27 选择的虚拟节点分别是 02/12/24/02，也就是真实节点 2/2/4/2。
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}
	
	//验证哈希分配:遍历 testCases，使用 hash.Get(k) 获取键 k 应该路由到的节点，并验证是否与预期的节点 v 匹配。
	for k, v := range testCases {
		if hash.GetPeer(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// Adds 8, 18, 28
	hash.Register("8") //添加一个新节点

	// 27 should now map to 8.
	//添加一个真实节点 8，对应虚拟节点的哈希值是 08/18/28，此时，用例 27 对应的虚拟节点从 02 变更为 28，即真实节点 8。
	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.GetPeer(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
