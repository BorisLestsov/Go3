package main

import (
    "fmt"
    "time"
    "net"
    "strconv"
    "strings"
    msg "./message"
)


func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func updateProcList(list *string, id int){
    idS := strconv.Itoa(id)
    strList := strings.Split(*list, "@")

    if !stringInSlice(idS, strList) {
        *list = *list + "@"+idS
    }
} 

func maxProcID(list string) int {
    strList := strings.Split(list, "@")
    var max int
    for _, e := range strList {
        val, err := strconv.Atoi(e)
        msg.CheckError(err)
        if val > max {
            max = val
        }
    }
    return max
} 


func isProcInList(list string, procID int) bool {
    strList := strings.Split(list, "@")
    for _, e := range strList {
        val, err := strconv.Atoi(e)
        msg.CheckError(err)
        if val == procID {
            return true
        }
    }
    return false
} 




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
          NProc int,
          PortArr []int,
          MaintArr []int, 
          quitCh chan struct{}) {

    //MyID := i
    MyPort := PortArr[MyID]
    MyMaintPort := MaintArr[MyID]
    LeftID := ((MyID-1)%NProc+NProc)%NProc
    RightID := ((MyID+1)%NProc+NProc)%NProc
    LeftPort  := PortArr[LeftID]
    RightPort  := PortArr[RightID]

    MyAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyPort))
    msg.CheckError(err)
    MyMaintAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyMaintPort))
    msg.CheckError(err)

    LeftAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(LeftPort))
    msg.CheckError(err)

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

    go ManageConn(MyConn, dataCh, time.Second * time.Duration(NProc) * 2)
    go ManageConn(MyMaintConn, maintCh, time.Second * 0)

    buffer := make([]byte, 4096)
    var m msg.Message
    var needDrop bool
    needDrop = false
    var terminated bool
    terminated = false

    var noToken bool
    noToken = false
    var generated bool
    generated = false
    var lastElectMsg msg.Message
    electDoubled := make([]bool, NProc)
    for i := range electDoubled {
        electDoubled[i] = false
    }
 
    var msgReturned bool
    msgReturned = false

    for {
        select {
            case m = <- dataCh: {if terminated {
                fmt.Println(m)
                continue
                
                }
            }
            case m = <- maintCh: {}
        }
        if m.Type_ == "token" {
            // Ordinary message
            if !noToken { // discard old token if nesessary
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
                needDrop = false
            } else {
                fmt.Println(MyID, " discarded token")
            }
        } else if m.Type_ == "timeout" {
            if !noToken {
                // timeout, initialize election
                fmt.Println("node", 
                            MyID, 
                            ": received timeout, launching election")
                

                //time.Sleep(time.Millisecond * 1000)
                noToken = true

                elect_m := msg.DataMessage{Type_: "elect", Dst_: RightID, Src_: MyID, Data_: strconv.Itoa(MyID)}
                m = msg.Message{Type_: "elect", Dst_: MyID, Data_: string(elect_m.ToJsonDataMsg())}
                lastElectMsg = m

                buffer = m.ToJsonMsg()
                time.Sleep(time.Millisecond * 1000)
                _, err = MyConn.WriteToUDP(buffer, RightAddr)
                msg.CheckError(err)

                elect_m = msg.DataMessage{Type_: "elect", Dst_: LeftID, Src_: MyID, Data_: strconv.Itoa(MyID)}
                m = msg.Message{Type_: "elect", Dst_: MyID, Data_: string(elect_m.ToJsonDataMsg())}
                lastElectMsg = m

                buffer = m.ToJsonMsg()
                time.Sleep(time.Millisecond * 1000)
                _, err = MyConn.WriteToUDP(buffer, LeftAddr)
                msg.CheckError(err)

                for i := range electDoubled {
                    electDoubled[i] = false
                }
                electDoubled[MyID] = true
                msgReturned = false
                generated = false
            } else {
                // timeout after initialized election => some proc died
                fmt.Println("node", MyID, ": some died")
                m_res := msg.FromJsonDataMsg(([]byte)(lastElectMsg.Data_))
                max := maxProcID(m_res.Data_)
                for i := 0; i < NProc; i++ {
                    if !isProcInList(m_res.Data_, LeftID) {
                        LeftID = ((LeftID-1)%NProc+NProc)%NProc
                        LeftPort = PortArr[LeftID]
                        LeftAddr,err = net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(LeftPort))
                        msg.CheckError(err)
                        NProc -= 1
                    }
                    if !isProcInList(m_res.Data_, RightID) {
                        RightID = ((RightID+1)%NProc+NProc)%NProc
                        RightPort = PortArr[RightID]
                        RightAddr,err = net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(RightPort))
                        msg.CheckError(err)
                        NProc -= 1
                    }
                }
                if max == MyID && noToken {
                    fmt.Println("node", 
                                MyID, 
                                ": generated token wow")
                    // generate new token
                    m_tmp := msg.DataMessage{Type_: "empty", Dst_: -1, Data_: ""}
                    m = msg.Message{Type_: "token", Dst_: -1, Data_: string(m_tmp.ToJsonDataMsg())}
                    buffer = m.ToJsonMsg()

                    time.Sleep(time.Millisecond * 1000)
                    _,err := MyConn.WriteToUDP(buffer, RightAddr)
                    msg.CheckError(err)
                } else {
                    time.Sleep(time.Duration(NProc))
                }

                noToken = false
            }
        } else if m.Type_ == "elect" {
            lastElectMsg = m
            if m.Dst_ == MyID {
                if !msgReturned {
                    m_tmp := msg.FromJsonDataMsg(([]byte)(m.Data_))
                    msgReturned = true
                    for i := 0; i < NProc; i++ {
                        if !isProcInList(m_tmp.Data_, i) {
                            msgReturned = false
                            break
                        }
                    }
                }
                if msgReturned {
                    // election token came back
                    m = msg.Message{Type_: "electfin", Dst_: MyID, Data_: m.Data_}

                    buffer = m.ToJsonMsg()
                    time.Sleep(time.Millisecond * 1000)
                    _, err = MyConn.WriteToUDP(buffer, RightAddr)
                    msg.CheckError(err)
                }
            } else {
                // foreign elect token, pass further updated token
                fmt.Println("node", 
                            MyID, 
                            ": received election token", 
                            "with data:",
                            m)
                if !generated {
                    noToken = true
                }

                if !electDoubled[m.Dst_] {
                    // duplicate elect token
                     //fmt.Println()
                    m_tmp := msg.FromJsonDataMsg([]byte(m.Data_))
                    m_tmp.Src_ = MyID
                    m_tmp.Dst_ = RightID
                    updateProcList(&m_tmp.Data_, MyID)
                    m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

                    lastElectMsg = m

                    buffer = m.ToJsonMsg()
                    time.Sleep(time.Millisecond * 1000)
                    _, err = MyConn.WriteToUDP(buffer, RightAddr)
                    msg.CheckError(err)

                    m_tmp.Dst_ = LeftID
                    m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

                    buffer = m.ToJsonMsg()
                    time.Sleep(time.Millisecond * 1000)
                    _, err = MyConn.WriteToUDP(buffer, LeftAddr)
                    msg.CheckError(err)

                    electDoubled[m.Dst_] = true
                } else {
                    // pass elect token further
                    m_tmp := msg.FromJsonDataMsg([]byte(m.Data_))
                    
                    if m_tmp.Src_ == LeftID {                        
                        m_tmp.Src_ = MyID
                        m_tmp.Dst_ = RightID
                        updateProcList(&m_tmp.Data_, MyID)
                        m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

                        lastElectMsg = m

                        buffer = m.ToJsonMsg()
                        time.Sleep(time.Millisecond * 1000)
                        _, err = MyConn.WriteToUDP(buffer, RightAddr)
                        msg.CheckError(err)
                    } else {
                        m_tmp.Src_ = MyID
                        m_tmp.Dst_ = LeftID
                        updateProcList(&m_tmp.Data_, MyID)
                        m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

                        lastElectMsg = m

                        buffer = m.ToJsonMsg()
                        time.Sleep(time.Millisecond * 1000)
                        _, err = MyConn.WriteToUDP(buffer, LeftAddr)
                        msg.CheckError(err)
                    }
                }
            }
        } else if m.Type_ == "electfin" {
            // second round
            if noToken {
                msgReturned = true
                fmt.Println("node", 
                            MyID, 
                            ": received election token", 
                            "with data:",
                            m)
                lastElectMsg = m
                if m.Dst_ != MyID {
                    // pass electfin token further
                    buffer = m.ToJsonMsg()
                    time.Sleep(time.Millisecond * 1000)
                    _, err = MyConn.WriteToUDP(buffer, RightAddr)
                    msg.CheckError(err)
                }
                max := maxProcID(msg.FromJsonDataMsg(([]byte)(lastElectMsg.Data_)).Data_)
                if max == MyID && !generated {
                    fmt.Println("node", 
                                MyID, 
                                ": generated token")
                    // generate new token
                    m_tmp := msg.DataMessage{Type_: "empty", Dst_: -1, Data_: ""}
                    m = msg.Message{Type_: "token", Dst_: -1, Data_: string(m_tmp.ToJsonDataMsg())}
                    buffer = m.ToJsonMsg()

                    time.Sleep(time.Millisecond * 1000)
                    _,err := MyConn.WriteToUDP(buffer, RightAddr)
                    msg.CheckError(err)
                    generated = true
                }
                noToken = false
            }
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
                    terminated = true                    
                case "recover":
                    terminated = false
                case "drop":
                    taskCh <- m
                default: 
                    fmt.Println("WTF")
            }
        }

        // finally send the message from queue

        // if !needDrop { 
        //     buffer = m.ToJsonMsg()

        //     time.Sleep(time.Millisecond * 1000)
        //     _, err = MyConn.WriteToUDP(buffer, RightAddr)
        //     msg.CheckError(err)
        // }
        // needDrop = false
    }

    <-quitCh
    //<-quitCh
    //MyConn.Close()
    //MyMaintConn.Close()
    quitCh <- struct{}{}
}
