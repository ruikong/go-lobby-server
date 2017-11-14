package main

import (
	"fmt"
	"encoding/binary"
	"bytes"
	"hash/crc32"  
)

type Packet struct {  
    length uint32  
    crc32  uint32
    info   string
}

func (p Packet) Encode() []byte {  
    buf2 := new(bytes.Buffer)  
    var length int = len([]byte(p.info))  
    err := binary.Write(buf2, binary.LittleEndian, (int32)(length))  
    CheckError(err)  
  
    err = binary.Write(buf2, binary.LittleEndian, []byte(p.info))  
    CheckError(err)  
  
    buf := new(bytes.Buffer)  
    p.length = uint32(buf2.Len() + 8)  
    err = binary.Write(buf, binary.LittleEndian, p.length)  
    CheckError(err)  
  
    p.crc32 = crc32.ChecksumIEEE(buf2.Bytes())  
    err = binary.Write(buf, binary.LittleEndian, p.crc32)
    CheckError(err)
  
    err = binary.Write(buf, binary.LittleEndian, buf2.Bytes())  
    CheckError(err)
    return buf.Bytes()
}  
  
func (p *Packet) Decode(buff []byte) {  
    buf := bytes.NewBuffer(buff)  
    err := binary.Read(buf, binary.LittleEndian, &(p.length))  
    CheckError(err)  
    fmt.Println(p.length)  
  
    err = binary.Read(buf, binary.LittleEndian, &(p.crc32))
    CheckError(err)
  
    buf2 := bytes.NewBuffer(buff[8:])
    crc := crc32.ChecksumIEEE(buf2.Bytes())
    if crc != p.crc32 {
        fmt.Errorf(" crc not check")  
    }
    p.info = (string)(buf2.Bytes())
    fmt.Printf("%s", p.info)
}  
