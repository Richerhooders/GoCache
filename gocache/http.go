package gocache

import (
	"fmt"
	"gocache/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_gocache/"
	defaultReplicas = 50
)
// HTTPPool implements PeerPicker for a pool of HTTP peers.
// self,用来记录自己的地址，包括主机名/IP和端口
// basePath, 作为节点间通讯地址的前缀，默认是/_gocache/,那么 http://example.com/_gocache/ 开头的请求，就用于节点间的访问
// 因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API 接口，一般以 /api 作为前缀
type HTTPPool struct {
	//this peer's base URL, e.g. "https://example.net:8000"
	self	string
	basePath string
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
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {  //判断访问路径的前缀是否是 basePath
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	// 请求的路径应该遵循格式 /<basePath>/<groupName>/<key>。通过 strings.SplitN 函数将路径分解成两部分
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
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
	//w.Write() 将缓存值作为 httpResponse 的 body 返回。
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

//创建具体的 HTTP 客户端类 httpGetter，实现 PeerGetter 接口。
type httpGetter struct {
	baseURL string
}

func(h *httpGetter)Get(group string,key string) ([]byte,error) {
	//baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_gocache/
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	//http.Get() 方式获取返回值，并转换为 []bytes 类型。
	res, err := http.Get(u)
	if err != nil {
		return nil,err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	return bytes, nil
}