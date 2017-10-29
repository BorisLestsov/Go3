package main

import (
    "strconv"
    "net"
    msg "./message"
)


func main() {
    var (
        FromPort = 60000
        ToPort = 40000


        )

    MyAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(FromPort))
    msg.CheckError(err)
    MyConn, err := net.ListenUDP("udp", MyAddr)
    msg.CheckError(err)

    ToAddr, err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(ToPort))
    msg.CheckError(err)


    data := "hello"
    buffer := make([]byte, 4096)

    m := msg.Message{Type_: "terminate", Dst_: 2, Data_: data}
    buffer = m.ToJsonMsg()

    _,err = MyConn.WriteToUDP(buffer, ToAddr)
    msg.CheckError(err)

}
