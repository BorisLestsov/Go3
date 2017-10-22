package main

import (
    "fmt"
    "time"
    "net"
    "strconv"
    msg "./message"
)


func ManageConn(MyID int,
                Conn *net.UDPConn,
                MyAddr *net.UDPAddr,
                LeftID int, 
                LeftAddr *net.UDPAddr, 
                RightID int,
                RightAddr *net.UDPAddr, 
                quitCh chan struct{}) {
    var data string
    var buffer = make([]byte, 8192)
    var m msg.Message

    for {
        n,addr,err := Conn.ReadFromUDP(buffer)
        msg.CheckError(err)
        data = string(buffer[0:n])
        fmt.Println(MyAddr, "Received ", data, " from ", addr)

        m = msg.FromJson(buffer[0:n])
        
        if m.Msgtype == 0 {
            // Ordinary message
            if m.ToID == MyID {
                fmt.Println(MyAddr, "Mine!", data, " from ", addr)
                m = msg.Message{TokenID: m.TokenID, Msgtype: 0, FromID: MyID, ToID: RightID, Data: "1"}

                buffer = m.ToJson()

                time.Sleep(time.Millisecond * 1000)

                _, err = Conn.WriteToUDP(buffer, RightAddr)
                msg.CheckError(err)

                //quitCh <- struct{}{}
                //return
            } else {
                buffer = m.ToJson()

                time.Sleep(time.Millisecond * 1000)
                _, err = Conn.WriteToUDP(buffer, RightAddr)
                msg.CheckError(err)
            }
        } else {
            //Maintance message
            switch m.Msgtype{
                case 1:
                    fmt.Println(MyAddr, "1!", data, " from ", addr)
                default:
                    fmt.Println(MyAddr, "EPTA!", data, " from ", addr)
            }
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

    if  MyID == 0 {
    	// init transmission
        data := "hello"
        buffer := make([]byte, 4096)

        m := msg.Message{TokenID: 0, Msgtype: 0, FromID: MyID, ToID: 8, Data: data}
        buffer = m.ToJson()

        _,err := MyConn.WriteToUDP(buffer, RightAddr)
        msg.CheckError(err)
        //_,err = MyConn.WriteToUDP(buffer, LeftAddr)
        //msg.CheckError(err)

        // m = msg.Message{Id: MyID, Msgtype: 1, Sender: MyID, Origin: MyID, Data: data}
        // buffer = m.ToJson()

        // _,err = MyMaintConn.WriteToUDP(buffer, RightMaintAddr)
        // msg.CheckError(err)
        // _,err = MyMaintConn.WriteToUDP(buffer, LeftMaintAddr)
        // msg.CheckError(err)


    }

    _,_,_,_,_,_,_,_ = MyID, MyMaintConn, MyMaintAddr, LeftID, LeftMaintAddr, RightID, RightMaintAddr, quitCh

    go ManageConn(MyID, MyConn, MyAddr, LeftID, LeftAddr, RightID, RightAddr, quitCh)
    go ManageConn(MyID, MyMaintConn, MyMaintAddr, LeftID, LeftMaintAddr, RightID, RightMaintAddr, quitCh)

    <-quitCh
    //<-quitCh
    //MyConn.Close()
    //MyMaintConn.Close()
    quitCh <- struct{}{}
}
