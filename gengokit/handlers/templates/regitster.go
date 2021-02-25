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
	"os"
	"strconv"
	"strings"
	"time"
)

const serviceName = "{{.ImportPath -}}"
const logDir = "./nacos/log"
const cacheDir = "./nacos/cache"

var reg *register

func Register() *register {
	if reg == nil {
		reg = &register{}
		if os.Getenv("SERVICE.REGISTER") == "true" {
			reg.parseServer()
			reg.naClient = reg.nacosClient()
		}
	}
	return reg
}

type register struct {
	naClient naming_client.INamingClient
	httpConf *serverConf
	grpcConf *serverConf
}

// 注册服务
func (r *register) Up() {
	if os.Getenv("SERVICE.REGISTER") != "true" {
		return
	}
	// 延迟10秒注册服务
	timeAfterTrigger := time.After(time.Second * 1)
	currTime, _ := <-timeAfterTrigger
	fmt.Printf("register server %s in %s\n", serviceName, currTime.Format("2006-01-02 15:04:05"))

	if r.grpcConf != nil {
		success, err := r.naClient.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          r.grpcConf.ServerIP,
			Port:        r.grpcConf.ServerPort,
			ServiceName: r.grpcConf.ServerName,
			Weight:      10,
			Enable:      true,
			Healthy:     true,
			Ephemeral:   true,
			Metadata: map[string]string{
				"ip":     r.grpcConf.ServerIP,
				"port":   fmt.Sprintf("%d", r.grpcConf.ServerPort),
				"source": r.grpcConf.Source,
				"mode":   "grpc",
			},
			GroupName: r.grpcConf.Namespace,
		})
		if err != nil {
			panic(err)
		}
		resStr := "failed"
		if success {
			resStr = "success"
		}
		fmt.Printf("register up serverName: %s %s\n", r.grpcConf.ServerName, resStr)
	}

	if r.httpConf != nil {
		success, err := r.naClient.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          r.httpConf.ServerIP,
			Port:        r.httpConf.ServerPort,
			ServiceName: r.httpConf.ServerName,
			Weight:      10,
			Enable:      true,
			Healthy:     true,
			Ephemeral:   true,
			Metadata: map[string]string{
				"ip":     r.httpConf.ServerIP,
				"port":   fmt.Sprintf("%d", r.httpConf.ServerPort),
				"source": r.httpConf.Source,
				"mode":   "http",
			},
			GroupName: r.httpConf.Namespace,
		})
		if err != nil {
			panic(err)
		}
		resStr := "failed"
		if success {
			resStr = "success"
		}
		fmt.Printf("register up serverName: %s %s\n", r.httpConf.ServerName, resStr)
	}

}

// 注销服务
func (r *register) Down() {
	if os.Getenv("SERVICE.REGISTER") != "true" {
		return
	}
	if r.grpcConf != nil {
		success, err := r.naClient.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          r.grpcConf.ServerIP,
			Port:        r.grpcConf.ServerPort,
			ServiceName: r.grpcConf.ServerName,
			GroupName:   r.grpcConf.Namespace,
			Ephemeral:   true,
		})
		if err != nil {
			panic(err)
		}
		resStr := "failed"
		if success {
			resStr = "success"
		}
		fmt.Printf("register down serverName: %s %s\n", r.grpcConf.ServerName, resStr)
	}

	if r.httpConf != nil {
		success, err := r.naClient.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          r.httpConf.ServerIP,
			Port:        r.httpConf.ServerPort,
			ServiceName: r.httpConf.ServerName,
			GroupName:   r.httpConf.Namespace,
			Ephemeral:   true,
		})
		if err != nil {
			panic(err)
		}
		resStr := "failed"
		if success {
			resStr = "success"
		}
		fmt.Printf("register down serverName: %s %s\n", r.httpConf.ServerName, resStr)
	}
}

// 获得nacos客户端
func (r *register) nacosClient() naming_client.INamingClient {
	clientConf := r.getNacosClientConf()
	serverConfs := r.getNacosServerConf()
	namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &clientConf,
		ServerConfigs: serverConfs,
	})
	if err != nil {
		panic(err)
	}
	return namingClient
}

// 获取 nacos 客户端配置
func (r *register) getNacosClientConf() constant.ClientConfig {
	nacf := constant.ClientConfig{
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              logDir,
		CacheDir:            cacheDir,
		RotateTime:          "24h",
		MaxAge:              3,
		LogLevel:            "debug",
	}

	if npid := os.Getenv("NACOS.REGISTER.CLIENT.NamespaceId"); npid != "" {
		nacf.NamespaceId = npid
	} else if r.grpcConf != nil {
		nacf.NamespaceId = r.grpcConf.NameSpaceId
	} else if r.httpConf != nil {
		nacf.NamespaceId = r.httpConf.NameSpaceId
	}

	if timeoutMs := os.Getenv("NACOS.REGISTER.CLIENT.TimeoutMs"); timeoutMs != "" {
		ms, err := strconv.ParseUint(timeoutMs, 10, 64)
		if err != nil {
			panic(err)
		}
		nacf.TimeoutMs = ms
	}

	if useCache := os.Getenv("NACOS.REGISTER.CLIENT.NotLoadCacheAtStart"); useCache == "false" {
		nacf.NotLoadCacheAtStart = false
	}

	if logDir := os.Getenv("NACOS.REGISTER.CLIENT.LogDir"); logDir != "" {
		nacf.LogDir = logDir
	}

	if cacheDir := os.Getenv("NACOS.REGISTER.CLIENT.CacheDir"); cacheDir != "" {
		nacf.CacheDir = cacheDir
	}

	if rotateTime := os.Getenv("NACOS.REGISTER.CLIENT.RotateTime"); rotateTime != "" {
		nacf.RotateTime = rotateTime
	}

	return nacf
}

func (r *register) getNacosServerConf() []constant.ServerConfig {
	cfs := make([]constant.ServerConfig, 0, 3)
	cf := constant.ServerConfig{
		IpAddr:      "127.0.0.1",
		ContextPath: "/nacos",
		Port:        8848,
		Scheme:      "http",
	}
	if ip := os.Getenv("NACOS.REGISTER.SERVER.IP"); ip != "" {
		cf.IpAddr = ip
	}
	if portStr := os.Getenv("NACOS.REGISTER.SERVER.Port"); portStr != "" {
		port, err := strconv.ParseUint(portStr, 10, 64)
		if err != nil {
			panic(err)
		}
		cf.Port = port
	}
	if contextPath := os.Getenv("NACOS.REGISTER.SERVER.ContextPath"); contextPath != "" {
		cf.ContextPath = contextPath
	}
	if scheme := os.Getenv("NACOS.REGISTER.SERVER.Scheme"); scheme != "" {
		cf.Scheme = scheme
	}
	cfs = append(cfs, cf)
	return cfs
}

type serverConf struct {
	Namespace   string ` + "`" + `json:"namespace"` + "`" + `
	NameSpaceId string ` + "`" + `json:"name_space_id"` + "`" + `
	ServerName  string ` + "`" + `json:"server_name"` + "`" + `
	ServerIP    string ` + "`" + `json:"server_ip"` + "`" + `
	ServerPort  uint64 ` + "`" + `json:"server_port"` + "`" + `
	Source      string ` + "`" + `json:"source"` + "`" + `
}

type namespaceResponse struct {
	Code    int            ` + "`" + `json:"code"` + "`" + `
	Message string         ` + "`" + `json:"message"` + "`" + `
	Data    []namespaceIns ` + "`" + `json:"data"` + "`" + `
}

type namespaceIns struct {
	Namespace         string ` + "`" + `json:"namespace"` + "`" + `
	NamespaceShowName string ` + "`" + `json:"namespaceShowName"` + "`" + `
	Quota             int    ` + "`" + `json:"quota"` + "`" + `
	ConfigCount       int    ` + "`" + `json:"configCount"` + "`" + `
	Type              int    ` + "`" + `json:"type"` + "`" + `
}

func (r *register) parseServer() {
	serverSplice := strings.Split(serviceName, "-")
	if len(serverSplice) < 4 {
		panic(fmt.Errorf("serverName incorrect format: %s", serviceName))
	}
	namespace := serverSplice[0]
	namespaceID := r.getNamespaceID(namespace)
	baseServerName := strings.Join(serverSplice[1:len(serverSplice)-1], ".")
	if enable := os.Getenv("SERVICE.HTTP.ENABLE"); enable != "false" { //  http service
		r.httpConf = &serverConf{
			Namespace:   namespace,
			NameSpaceId: namespaceID,
			ServerName:  baseServerName + ".http",
			ServerIP:    r.getServerIP("HTTP"),
			ServerPort:  r.getServerPort("HTTP"),
			Source:      serviceName,
		}
	}
	if enable := os.Getenv("SERVICE.GRPC.ENABLE"); enable != "false" { //  grpc service
		r.grpcConf = &serverConf{
			Namespace:   namespace,
			NameSpaceId: namespaceID,
			ServerName:  baseServerName + ".grpc",
			ServerIP:    r.getServerIP("GRPC"),
			ServerPort:  r.getServerPort("GRPC"),
			Source:      serviceName,
		}
	}
}

// 查询命名空间ID
func (r *register) getNamespaceID(namespaceShowName string) string {
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

func (r *register) getServerIP(mode string) string {
	ip := os.Getenv(fmt.Sprintf("SERVICE.%s.IP", strings.ToUpper(mode)))
	if ip != "" {
		return ip
	}
	// get system ip address
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

func (r *register) getServerPort(mode string) uint64 {
	portStr := os.Getenv(fmt.Sprintf("SERVICE.%s.PORT", strings.ToUpper(mode)))
	if portStr == "" {
		portStr = "9090"
	}
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		panic(err)
	}
	return port
}

`
