package main

import (
	"fmt"
	"net"
	"net/http"
	"encoding/json"
	"encoding/binary"
	"bytes"
)

var serverMap = make(map[string]*Server)

type Server struct {
	load int32
	port int32
	ip []byte
}

func (this *Server)Deserialize(buff []byte){
	pBuffer := bytes.NewBuffer(buff)
	err := binary.Read(pBuffer, binary.LittleEndian, &(this.load))
	CheckError(err)
	err = binary.Read(pBuffer, binary.LittleEndian, &(this.port))
	CheckError(err)
	this.ip = pBuffer.Bytes()
	CheckError(err)
}

func (this *Server)Serialize() {
	buffer := new(bytes.Buffer) 
    err := binary.Write(buffer, binary.LittleEndian, this.load)  
	CheckError(err)
    err = binary.Write(buffer, binary.LittleEndian, this.port)  
	CheckError(err)
	err = binary.Write(buffer, binary.LittleEndian, this.ip)
	CheckError(err)
}

type ApiWapper struct {
	code int
	msg string
	data interface{}
}

func main() {
	startTcpServer("localhost", 8090)
}

func startTcpServer(ip string, port int){
	netListen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	CheckError( err )
	defer netListen.Close()
	fmt.Print("Waiting for clients")
	for {
		conn, err := netListen.Accept()
		if err != nil  {
			continue
		}
		fmt.Print(conn.RemoteAddr().String(), " tcp connect success")
	}
}

func startHttpServer(ip string, port int) {
	http.HandleFunc("/servers", handleFetchAvailableServer)
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

		fmt.Printf(conn.RemoteAddr().String(), "receive data string :\n", string(buffer[:n]))

		handleTcpMessage(buffer[:n], n)
	}
}

func handleTcpMessage(buffer []byte, length int) {
	pack := &Packet{}
	pack.Decode(buffer)
	server := &Server{}
	server.Deserialize(pack.data)
	fmt.Printf("receive msg [ip:%s port:%d load:%d]", server.ip, server.port, server.load)
}

func handleFetchAvailableServer(w http.ResponseWriter, req *http.Request) {
	api := &ApiWapper{0,"success",nil}
	jsonbyte, err := json.Marshal(api)
	CheckError(err)
	w.Write(jsonbyte)
}

func CheckError(err error) {
	fmt.Print(err)
}