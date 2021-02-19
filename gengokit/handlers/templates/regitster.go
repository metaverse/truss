package templates

const Register = `
package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

const server = "{{.ImportPath -}}"
const logDir = "./nacos/log"
const cacheDir = "./nacos/cache"

func Register() *register {
	rg := &register{}
	rg.serverConf = parseServer()
	rg.naClient = rg.nacosClient()
	return rg
}

type register struct {
	naClient   naming_client.INamingClient
	serverConf *serverConf
}

// 注册服务
func (r *register) Up() {
	// 延迟10秒注册服务
	timeAfterTrigger := time.After(time.Second * 1)
	currTime, _ := <-timeAfterTrigger
	fmt.Printf("register server %s in %s\n", r.serverConf.ServerName, currTime.Format("2006-01-02 15:04:05"))
	success, err := r.naClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          r.serverConf.ServerIP,
		Port:        r.serverConf.ServerPort,
		ServiceName: r.serverConf.ServerName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"test": "registerTest"},
		GroupName: r.serverConf.Namespace,
	})
	if err != nil {
		panic(err)
	}
	resStr := "failed"
	if success {
		resStr = "success"
	}
	fmt.Printf("register up serverName: %s %s\n", r.serverConf.ServerName, resStr)
}

// 注销服务
func (r *register) Down() {
	success, err := r.naClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          r.serverConf.ServerIP,
		Port:        r.serverConf.ServerPort,
		ServiceName: r.serverConf.ServerName,
		GroupName:   r.serverConf.Namespace,
		Ephemeral:   true,
	})
	if err != nil {
		panic(err)
	}
	resStr := "failed"
	if success {
		resStr = "success"
	}
	fmt.Printf("register down serverName: %s %s\n", r.serverConf.ServerName, resStr)
}

// 获得nacos客户端
func (r *register) nacosClient() naming_client.INamingClient {
	clientConf := constant.ClientConfig{
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              logDir,
		CacheDir:            cacheDir,
		RotateTime:          "24h",
		MaxAge:              3,
		LogLevel:            "debug",
	}
	if r.serverConf.NameSpaceId != "" {
		clientConf.NamespaceId = r.serverConf.NameSpaceId
	}
	serverConfs := []constant.ServerConfig{
		{
			IpAddr:      "127.0.0.1",
			ContextPath: "/nacos",
			Port:        8848,
			Scheme:      "http",
		},
	}
	namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &clientConf,
		ServerConfigs: serverConfs,
	})
	if err != nil {
		panic(err)
	}
	return namingClient
}

type serverConf struct {
	Namespace   string
	NameSpaceId string
	ServerName  string
	ServerIP    string
	ServerPort  uint64
	Source      string
}

type namespaceIns struct {
	Namespace         string ` + "`" + `json:"namespace"` + "`" + `
	NamespaceShowName string ` + "`" + `json:"namespaceShowName"` + "`" + `
	Quota             int    ` + "`" + `json:"quota"` + "`" + `
	ConfigCount       int    ` + "`" + `json:"configCount"` + "`" + `
	Type              int    ` + "`" + `json:"type"` + "`" + `
}

type namespaceResponse struct {
	Code    int            ` + "`" + `json:"code"` + "`" + `
	Message string         ` + "`" + `json:"message"` + "`" + `
	Data    []namespaceIns ` + "`" + `json:"data"` + "`" + `
}

func parseServer() *serverConf {
	serverSplice := strings.Split(server, "-")
	if len(serverSplice) < 4 {
		panic(fmt.Errorf("serverName incorrect format: %s", server))
	}
	conf := &serverConf{
		Namespace:   serverSplice[0],
		NameSpaceId: getNamespaceID(serverSplice[0]),
		ServerName:  strings.Join(serverSplice[1:len(serverSplice)-1], "."),
		ServerIP:    getServerIP(),
		ServerPort:  5050,
		Source:      server,
	}
	return conf
}

// 查询命名空间ID
func getNamespaceID(namespaceShowName string) string {
	apiUrl := fmt.Sprintf("http://%s:%d/nacos/v1/console/namespaces", "127.0.0.1", 8848)
	client := http.Client{}
	client.Timeout = time.Microsecond * time.Duration(5000)
	request, err := http.NewRequest(http.MethodGet, apiUrl, nil)
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resData namespaceResponse
	if err := json.Unmarshal(body, &resData); err != nil {
		panic(err)
	}
	if resData.Code != 200 {
		panic(fmt.Errorf("namespaceID 解析错误，err:%s", resData.Message))
	}
	for _, cf := range resData.Data {
		if cf.NamespaceShowName == namespaceShowName {
			return cf.Namespace
		}
	}
	return ""
}

func getServerIP() string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, address := range addresses {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	panic(fmt.Errorf("not found valid IP address"))
}
`
