package gocache

import (
	"fmt"
	"gocache/consistenthash"
	pb "gocache/gocachepb"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
)

const (
	defaultBasePath = "/_gocache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
// 因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API 接口，一般以 /api 作为前缀
type HTTPPool struct {
	//this peer's base URL, e.g. "https://example.net:8000"
	self	string		// self,用来记录自己的地址，包括主机名/IP和端口
	basePath string		// basePath, 作为节点间通讯地址的前缀，默认是/_gocache/,那么 http://example.com/_gocache/ 开头的请求，就用于节点间的访问
	mu		sync.Mutex //guards peers and httpGetters
	peers	*consistenthash.Map    //一致性哈希算法的 Map，用来根据具体的 key 选择节点。

	//映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关。
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

//NewHTTPPool initializes an HTTP pool of peers
func NewHTTPPool(self string) *HTTPPool {
//http.ListenAndServe 接收 2 个参数，第一个参数是服务启动的地址
//第二个参数是 Handler，任何实现了 ServeHTTP 方法的对象都可以作为 HTTP 的 Handler。
	return &HTTPPool{
		self:	self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
// http.Hander接口要求实现ServerHTTP方法 http.ResponseWriter用于向客户端返回响应,*http.Requeset包含了客户端请求的所有信息，如URL，头信息和请求体
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {  //判断访问路径的前缀是否是 basePath
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	// 请求的路径应该遵循格式 /<basePath>/<groupName>/<key>。通过 strings.SplitN 函数将路径分解成两部分
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2) //从p.basePath下一个字符开始分割
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	//其中 parts[0] 是组名(groupName)，parts[1] 是键(key)
	//通过 groupname 得到 group 实例，再使用 group.Get(key) 获取缓存数据。
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Write the value to the response body as a proto message
	//ServeHTTP() 中使用 proto.Marshal() 编码 HTTP 响应。
	body,err := proto.Marshal(&pb.Response{Value:view.ByteSlice()})
	if err != nil {
		http.Error(w,err.Error(),http.StatusInternalServerError) 
		return
	}
	//w.Write() 将缓存值作为 httpResponse 的 body 返回。
	w.Header().Set("Content-Type", "application/octet-stream")
	// w.Write(view.ByteSlice())
	w.Write(body)
}

//创建具体的 HTTP 客户端类 httpGetter，实现 PeerGetter 接口。
type httpGetter struct {
	baseURL string
}

// func(h *httpGetter)Get(group string,key string) ([]byte,error) {
func (h *httpGetter)Get(in *pb.Request,out *pb.Response) error {
	//baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_gocache/
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		// url.QueryEscape(group),
		// url.QueryEscape(key),
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	//http.Get() 方式获取返回值，并转换为 []bytes 类型。
	res, err := http.Get(u)
	if err != nil {
		// return nil,err
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// return nil, fmt.Errorf("server returned: %v", res.Status)
		return fmt.Errorf("server returned: %v", res.Status)
	}
	bytes, err := io.ReadAll(res.Body)  //转换为[]bytes类型
	if err != nil {
		// return nil, fmt.Errorf("reading response body: %v", err)
		return fmt.Errorf("reading response body: %v", err)
	}
	//使用proto.Unmarshal()解码HTTP响应
	if err = proto.Unmarshal(bytes,out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	// return bytes, nil 无需返回res，因为res已经被写入参数Response中
	return nil
}

//Set updates the pool's list of peers
//Set() 方法实例化了一致性哈希算法，并且添加了传入的节点。
func(p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas,nil)
	//传入真实节点构建虚拟节点
	p.peers.Add(peers...)
	//为每个节点创建了一个新的映射其中 *httpGetter,用于处理对应节点的HTTP请求
	p.httpGetters = make(map[string]*httpGetter,len(peers))
	//将每个节点与对应的httpGetter实例在映射中关联，选择节点，返回节点对应的 HTTP 客户端（这里是ip地址）。
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

//PickPeer picks a peer according to key
//PickerPeer() 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
func (p *HTTPPool) PickPeer(key string) (PeerGetter,bool) {
	p.mu.Lock();
	defer p.mu.Unlock()
	if peer := p.peers.Get(key);peer != "" && peer != p.self {
		p.Log("Pick peer %s",peer)
		return p.httpGetters[peer],true
	}
	return nil,false
}

var _ PeerPicker = (*HTTPPool)(nil)
//至此，HTTPPool 既具备了提供 HTTP 服务的能力，也具备了根据具体的 key，创建 HTTP 客户端从远程节点获取缓存值的能力。