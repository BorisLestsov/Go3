package main

import (
    "fmt"
    "time"
    "net"
    "strconv"
    msg "./message"
)


func ManageConn(Conn *net.UDPConn,
                dataCh chan msg.Message) {
    var buffer = make([]byte, 4096)
    var m msg.Message
    //var data string

    for {
        n,addr,err := Conn.ReadFromUDP(buffer)
        _ = addr
        msg.CheckError(err)

        //data := string(buffer[0:n])
        m = msg.FromJsonMsg(buffer[0:n])
        dataCh <- m   
    }

}


func proc(MyID int, 
		  MyPort int,
		  MyMaintPort int, 
		  LeftID int, 
		  LeftPort int, 
		  RightID int, 
		  RightPort int, 
		  quitCh chan struct{}) {
    MyAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyPort))
    msg.CheckError(err)
    MyMaintAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyMaintPort))
    msg.CheckError(err)
 
    //LeftAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(LeftPort))
    //msg.CheckError(err)

    RightAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(RightPort))
    msg.CheckError(err)

    MyConn, err := net.ListenUDP("udp", MyAddr)
    msg.CheckError(err)
    MyMaintConn, err := net.ListenUDP("udp", MyMaintAddr)
    msg.CheckError(err)

    if  MyID == 0 {
    	// init transmission
        buffer := make([]byte, 4096)

        init_data_m := msg.DataMessage{Type_: "empty", Dst_: -1, Data_: ""}
        data := string(init_data_m.ToJsonDataMsg())        
        m := msg.Message{Type_: "token", Dst_: -1, Data_: data}
        buffer = m.ToJsonMsg()

        _,err := MyConn.WriteToUDP(buffer, RightAddr)
        msg.CheckError(err)

    }

    dataCh  := make(chan msg.Message)
    maintCh := make(chan msg.Message)
    taskCh := make(chan msg.Message, 4096)

    go ManageConn(MyConn, dataCh)
    go ManageConn(MyMaintConn, maintCh)

    buffer := make([]byte, 4096)
    var m msg.Message

    for {
        select {
            case m = <- dataCh: {}
            case m = <- maintCh: {}
        }
        fmt.Println(MyAddr, "Received ", m)
        
        
        if m.Type_ == "token" {
            // Ordinary message
            m_data := msg.FromJsonDataMsg([]byte(m.Data_))
            if m.Dst_ == MyID {
                fmt.Println(MyAddr, "Mine!", m)
                switch m_data.Type_ {
                    case "conf": {
                        //got conformation, refresh token
                        init_data_m := msg.DataMessage{Type_: "empty", Dst_: -1, Data_: ""}
                        m = msg.Message{Type_: "token", Dst_: -1, Data_: string(init_data_m.ToJsonDataMsg())}
                    } 
                    case "send": {
                        // need to send conformation
                        conformation_m := msg.DataMessage{Type_: "conf", Dst_: m_data.Src_, Src_: MyID, Data_: ""}
                        m = msg.Message{Type_: "token", Dst_: m_data.Src_, Data_: string(conformation_m.ToJsonDataMsg())}
                    }
                }
                //quitCh <- struct{}{}
                //return
            } else if m.Dst_ == -1 {
                select {
                    case tmp := <-taskCh:
                        //we have unfulfilled maintance task
                        m_data := msg.DataMessage{Type_: "send", Dst_: tmp.Dst_, Src_: MyID, Data_: tmp.Data_}
                        m = msg.Message{Type_: "token", Dst_: tmp.Dst_, Data_: string(m_data.ToJsonDataMsg())}
                    default:
                        //pass token further
                }
            }
            buffer = m.ToJsonMsg()

            time.Sleep(time.Millisecond * 1000)
            _, err = MyConn.WriteToUDP(buffer, RightAddr)
            msg.CheckError(err)
        } else {
            //Maintance message
            switch m.Type_{
                case "send":
                    fmt.Println("send")
                    taskCh <- m
                case "terminate":
                    fmt.Println("terminate")
                case "recover":
                    fmt.Println("recover")
                case "drop":
                    fmt.Println("drop")
                default: 
                    fmt.Println("WTF")
            }
        }
    }

    <-quitCh
    //<-quitCh
    //MyConn.Close()
    //MyMaintConn.Close()
    quitCh <- struct{}{}
}
