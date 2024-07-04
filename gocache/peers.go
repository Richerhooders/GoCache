package gocache

// import pb "gocache/gocachepb"

/*
完成流程2
使用一致性哈希选择节点        是                                    是
    |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
                    |  否                                    ↓  否
                    |----------------------------> 回退到本地节点处理。
*/

//抽象出 2 个接口，PeerPicker 的 PickPeer() 方法用于根据传入的 key 选择相应节点 PeerGetter。
//PeerPicker is the interfact that must be implemented to locate
//the peer that owns a specific key
type Picker interface {
	Pick(key string) (peer Fetcher,ok bool)
}

// 接口 PeerGetter 的 Get() 方法用于从对应 group 查找缓存值。PeerGetter 就对应于上述流程中的 HTTP 客户端。
// PeerGetter is the interface that must be implemented by a peer.
// type PeerGetter interface {
// 	// Get(group string ,key string) ([]byte,error)
// 	Get(in *pb.Request,out *pb.Response) error
// }
// Fetcher 定义了从远端获取缓存的能力
// 所以每个Peer应实现这个接口
type Fetcher interface {
	Fetch(group string, key string) ([]byte, error)
}