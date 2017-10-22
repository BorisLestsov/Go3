package main

import (
    "fmt"
    //"time"
    "net"
    "strconv"
    msg "./message"
)


func ManageConn(MyID int,
                Conn *net.UDPConn, 
                MyAddr *net.UDPAddr, 
                LeftAddr *net.UDPAddr, 
                RightAddr *net.UDPAddr, 
                quitCh chan struct{}) {
    var data string
    var buffer = make([]byte, 8192)
    var m msg.Message
    var DestAddr *net.UDPAddr

    for {
        n,addr,err := Conn.ReadFromUDP(buffer)
        if err != nil {
            fmt.Println("Error: ",err)
        }
        data = string(buffer[0:n])
        fmt.Println(MyAddr, "Received ", data, " from ", addr)

        m = msg.FromJson(buffer[0:n])
        
        if addr == LeftAddr {
            DestAddr = RightAddr
        } else {
            DestAddr = LeftAddr
        }

        i, err := strconv.Atoi(m.Data)
        msg.CheckError(err)
        i -= 1
        
        m = msg.Message{Id: MyID, Msgtype: m.Msgtype, Sender: MyID, Origin: m.Origin, Data: strconv.Itoa(i)}
        buffer = m.ToJson()
        
        if i != 0 {
            data = string(buffer[0:n])
            _, err = Conn.WriteToUDP(buffer, DestAddr)
            msg.CheckError(err)
        } else {
            fmt.Println("i ==", i, "Exiting")
            quitCh <- struct{}{}
            return
        }
    }

}


func proc(MyID int, 
		  MyPort int,
		  MyMaintPort int, 
		  LeftID int, 
		  LeftPort int, 
		  LeftMaintPort int,
		  RightID int, 
		  RightPort int, 
		  RightMaintPort int, 
		  quitCh chan struct{}) {
    MyAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyPort))
    msg.CheckError(err)
    MyMaintAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyMaintPort))
    msg.CheckError(err)
 
    LeftAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(LeftPort))
    msg.CheckError(err)
    LeftMaintAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(LeftMaintPort))
    msg.CheckError(err)

    RightAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(RightPort))
    msg.CheckError(err)
    RightMaintAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(RightMaintPort))
    msg.CheckError(err)

    MyConn, err := net.ListenUDP("udp", MyAddr)
    msg.CheckError(err)
    MyMaintConn, err := net.ListenUDP("udp", MyMaintAddr)
    msg.CheckError(err)


    var isRoot bool 
    if MyID == 0 {
        isRoot = true
    }

    if isRoot {
    	// init transmission
        data := "10"
        buffer := make([]byte, 4096)

        m := msg.Message{Id: MyID, Msgtype: 0, Sender: MyID, Origin: MyID, Data: data}
        buffer = m.ToJson()

        _,err := MyConn.WriteToUDP(buffer, RightAddr)
        msg.CheckError(err)
        _,err = MyConn.WriteToUDP(buffer, LeftAddr)
        msg.CheckError(err)

        m = msg.Message{Id: MyID, Msgtype: 1, Sender: MyID, Origin: MyID, Data: data}
        buffer = m.ToJson()
        
        _,err = MyMaintConn.WriteToUDP(buffer, RightMaintAddr)
        msg.CheckError(err)
        _,err = MyMaintConn.WriteToUDP(buffer, LeftMaintAddr)
        msg.CheckError(err)


        isRoot = false
    }

    go ManageConn(MyID, MyConn, MyAddr, LeftAddr, RightAddr, quitCh)
    go ManageConn(MyID, MyMaintConn, MyMaintAddr, LeftMaintAddr, RightMaintAddr, quitCh)

    <-quitCh
    <-quitCh
    MyConn.Close()
    MyMaintConn.Close()
    quitCh <- struct{}{}
}
