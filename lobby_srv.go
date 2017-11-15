package main

import (
	"fmt"
	"net"
	"net/http"
	"encoding/json"
	"encoding/binary"
	"bytes"
	"sync"
	"time"
	"strings"
	"strconv"
)

func NewMap() *SafeMap {
    sm := new(SafeMap)
    sm.Map = make(map[string]*Server)
    return sm
}

type SafeMap struct {
    sync.RWMutex
    Map map[string]*Server
}

func (sm *SafeMap) readMap(key string) (*Server, bool){
    sm.RLock()
    value, exist := sm.Map[key]
    sm.RUnlock()
    return value, exist
}

func (sm *SafeMap) writeMap(key string, value *Server) {
    sm.Lock()
    sm.Map[key] = value
    sm.Unlock()
}

func (sm *SafeMap) count() int {
	count := 0
    sm.RLock()
	count = len(sm.Map)
	sm.RUnlock()
	return count
}

func (sm *SafeMap) Remove(key string) {
	sm.Lock()
    delete(sm.Map, key)
    sm.Unlock()
}

func (sm *SafeMap) ChooseServer() (*Server) {
	var srv *Server = nil
	if serverMap.count() == 0 {
		return nil
	}
	var load, i int32 = 0, 0
	sm.RLock()
    for _, s := range sm.Map {
		if i==0 {
			srv = s
			load = s.Load
		} else {
			if s.Load < load {
				srv = s
				load = s.Load
			}
		}
		i++
	}
	sm.RUnlock()
	return srv
}

func (sm *SafeMap) CheckServer() {
	sm.RLock()
	now := time.Now().Unix()
	arr := make([]string, 0)
    for k, s := range sm.Map {
		if s.Time < now - 60 {
			arr = append(arr, k)
		}
	}
	sm.RUnlock()

	for _, key := range arr {
		sm.Remove(key)
		fmt.Printf("删除服务列表:%s \n", key)
	}
}

var serverMap = NewMap()

type Server struct {
	Time int64
	Load int32
	Port int32
	Ip string
}

func (this *Server)Deserialize(buff []byte){
	pBuffer := bytes.NewBuffer(buff)
	err := binary.Read(pBuffer, binary.LittleEndian, &(this.Load))
	CheckError(err)
	err = binary.Read(pBuffer, binary.LittleEndian, &(this.Port))
	CheckError(err)
	this.Ip = string(pBuffer.Bytes())
	CheckError(err)
}

func (this *Server)Serialize() {
	buffer := new(bytes.Buffer) 
    err := binary.Write(buffer, binary.LittleEndian, this.Load)  
	CheckError(err)
    err = binary.Write(buffer, binary.LittleEndian, this.Port)  
	CheckError(err)
	err = binary.Write(buffer, binary.LittleEndian, this.Ip)
	CheckError(err)
}

type ApiWapper struct {
	Code int
	Msg string
	Data interface{}
}

func main() {
	go StartHttpServer("0.0.0.0", 8099)
	go StartTcpServer("0.0.0.0", 8088)

	StartTimerCheck()
}

func StartTimerCheck() {
	timer1 := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-timer1.C:
			CheckServerRunning()
		}
	}
}

func CheckServerRunning() {
	fmt.Printf("定时器检查服务列表\n")
	serverMap.CheckServer()
}

func StartTcpServer(ip string, port int){
	url := fmt.Sprintf("%s:%d", ip, port)
	netListen, err := net.Listen("tcp", url)
	CheckError( err )
	defer netListen.Close()

	fmt.Printf("服务监听成功 %s\n",url)

	for {
		conn, err := netListen.Accept()
		if err != nil  {
			continue
		}
		fmt.Print(conn.RemoteAddr().String(), " 客户端连接成功 \n")
	}
}

func StartHttpServer(ip string, port int) {
	fmt.Printf("Http服务器启动成功:%s:%d\n", ip, port)
	http.HandleFunc("/servers", HandleFetchAvailableServer)
	http.HandleFunc("/datas", HandleFetchData)
	http.HandleFunc("/servers/reg", HandleSrvReg)
	http.HandleFunc("/server/available", HandleSrvCheck)
	
	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)
}

func handleConnection(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read( buffer )
		if err != nil {
			fmt.Printf(conn.RemoteAddr().String(), "connection error :", err)
			return
		}

		fmt.Printf(conn.RemoteAddr().String(), " 收到客户端消息 \n")

		HandleTcpMessage(buffer[:n], n)
	}
}

func HandleTcpMessage(buffer []byte, length int) {
	pack := &Packet{}
	pack.Decode(buffer)
	srv := &Server{}
	srv.Deserialize(pack.data)

	HandleServer(srv)
}

func HandleServer(srv *Server) {
	key := fmt.Sprintf("%s:%d",srv.Ip, srv.Port)
	oldSrv, exist := serverMap.readMap(key)
	if exist {
		oldSrv.Load = srv.Load
		oldSrv.Time = time.Now().Unix()
	} else {
		srv.Time = time.Now().Unix()
		serverMap.writeMap(key, srv)
	}
	fmt.Printf("收到客户端消息 [time:%d, ip:%s port:%d load:%d]", srv.Time, srv.Ip, srv.Port, srv.Load)
}

func HandleFetchAvailableServer(w http.ResponseWriter, req *http.Request) {
	api := &ApiWapper{0,"success",nil}
	srv := serverMap.ChooseServer()
	if srv == nil {
		api.Code = 1
		api.Msg = "error"
	} else {
		api.Data = srv
	}
	jsonbyte, err := json.Marshal(api)
	CheckError(err)
	w.Write(jsonbyte)
}

func HandleFetchData(w http.ResponseWriter, req *http.Request){
	w.Write([]byte("{\"code\":0}"))
}

func HandleSrvReg(w http.ResponseWriter, req *http.Request){
	var ipStr = req.FormValue("ip")
	var portStr = req.FormValue("port")
	var loadStr = req.FormValue("load")
	api := &ApiWapper{1, "error",nil}

	for {
		if ipStr=="" || portStr=="" || loadStr=="" {
			api.Msg = "缺少参数"
			break
		}
		
		if len( strings.Split(ipStr, ".") ) != 4 {
			api.Msg = "IP错误"
			break
		}

		port, _ := strconv.Atoi(portStr)
		load, _ := strconv.Atoi(loadStr)
		srv := &Server{}
		srv.Ip = ipStr
		srv.Port = (int32)(port)
		srv.Load = (int32)(load)
		
		HandleServer(srv)

		api.Code = 0
		api.Msg = "success"

		break
	}

	jsonbyte, err := json.Marshal(api)
	CheckError(err)
	w.Write(jsonbyte)
}

func HandleSrvCheck(w http.ResponseWriter, req *http.Request){
	var ipStr = req.FormValue("ip")
	var portStr = req.FormValue("port")
	api := &ApiWapper{1, "error",nil}
	for {
		if ipStr=="" || portStr=="" {
			api.Msg = "缺少参数"
			break
		}
		
		if len( strings.Split(ipStr, ".") ) != 4 {
			api.Msg = "IP错误"
			break
		}
		key := fmt.Sprintf("%s:%s",ipStr,portStr)
		srv, isExist := serverMap.readMap(key)

		if !isExist {
			break
		}

		api.Code = 0
		api.Msg = "success"
		api.Data = srv
		
		break
	}

	jsonbyte, err := json.Marshal(api)
	CheckError(err)
	w.Write(jsonbyte)
}


func CheckError(err error) {
	if err != nil {
		fmt.Print(err)
	}
}