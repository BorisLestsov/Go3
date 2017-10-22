package main

import (
    "fmt"
    "time"
    "net"
    "strconv"
    msg "./message"
)


func ManageConn(Conn *net.UDPConn,
                dataCh chan msg.Message,
                timeout time.Duration) {
    var buffer = make([]byte, 4096)
    var m msg.Message
    //var data string

    
    for {
        if timeout != 0 {
            Conn.SetReadDeadline(time.Now().Add(timeout))
        }
        n,addr,err := Conn.ReadFromUDP(buffer)
        _ = addr
        if err != nil {
            if e, ok := err.(net.Error); !ok || !e.Timeout() {
                // not a timeout
                panic(err)
            } else {
                m = msg.Message{Type_: "timeout", Dst_: 0, Data_: ""}               
            }
        } else {
            //data := string(buffer[0:n])
            //fmt.Println(data)
            m = msg.FromJsonMsg(buffer[0:n])
        }
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

    go ManageConn(MyConn, dataCh, time.Second * 8)
    go ManageConn(MyMaintConn, maintCh, time.Second * 0)

    buffer := make([]byte, 4096)
    var m msg.Message
    var needDrop bool

    for {
        select {
            case m = <- dataCh: {}
            case m = <- maintCh: {}
        }

        if m.Type_ == "token" {
            // Ordinary message
            m_data := msg.FromJsonDataMsg([]byte(m.Data_))
            if m.Dst_ == MyID {
                switch m_data.Type_ {
                    case "conf": {
                        //got conformation, refresh token
                        fmt.Println("node", 
                                    MyID, 
                                    ": received token from node", 
                                    LeftID, 
                                    "with delivery confirmation from node", 
                                    m_data.Src_, 
                                    ", sending token to node", 
                                    RightID)
                        init_data_m := msg.DataMessage{Type_: "empty", Dst_: -1, Data_: ""}
                        m = msg.Message{Type_: "token", Dst_: -1, Data_: string(init_data_m.ToJsonDataMsg())}
                        
                    } 
                    case "send": {
                        fmt.Println("node", 
                                    MyID, 
                                    ": received token from node", 
                                    LeftID, 
                                    "with data from node", 
                                    m_data.Src_, 
                                    "(data =`", m_data.Data_, "`),",
                                    "sending token to node", 
                                    RightID)
                        // need to send conformation
                        conformation_m := msg.DataMessage{Type_: "conf", Dst_: m_data.Src_, Src_: MyID, Data_: ""}
                        m = msg.Message{Type_: "token", Dst_: m_data.Src_, Data_: string(conformation_m.ToJsonDataMsg())}
                    }
                }
            } else if m.Dst_ == -1 {
                // empty token
                select {
                    case tmp := <-taskCh:
                        //we have unfulfilled maintance task
                        switch tmp.Type_ {
                            case "send":
                                m_data := msg.DataMessage{Type_: "send", Dst_: tmp.Dst_, Src_: MyID, Data_: tmp.Data_}
                                m = msg.Message{Type_: "token", Dst_: tmp.Dst_, Data_: string(m_data.ToJsonDataMsg())}
                            case "terminate":
                                fmt.Println("terminate")
                            case "recover":
                                fmt.Println("recover")
                            case "drop":
                                needDrop = true
                            default: 
                                fmt.Println("Unknown maintance task!:", m.Type_)
                        }
                    default: {}
                }
                //pass empty token further
                fmt.Println("node", 
                            MyID, 
                            ": received token from node", 
                            LeftID, 
                            ", sending token to node", 
                            RightID)
            } else {
                //pass non empty token further
                fmt.Println("node", 
                            MyID, 
                            ": received token from node", 
                            LeftID, 
                            ", sending token to node", 
                            RightID)
            }
            if !needDrop { 
                buffer = m.ToJsonMsg()

                time.Sleep(time.Millisecond * 1000)
                _, err = MyConn.WriteToUDP(buffer, RightAddr)
                msg.CheckError(err)
            }
        } else if m.Type_ == "timeout" {
            fmt.Println("node", 
                        MyID, 
                        ": received timeout")
            time.Sleep(time.Millisecond * 1000)
        } else {
            //Maintance message
            fmt.Println("node", 
                        MyID, 
                        ": received service message:",
                        string(m.ToJsonMsg()))
            switch m.Type_{
                case "send":
                    taskCh <- m
                case "terminate":
                    fmt.Println("terminate")
                case "recover":
                    fmt.Println("recover")
                case "drop":
                    taskCh <- m
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
