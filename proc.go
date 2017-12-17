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
                timeoutCh chan time.Duration) {
    var buffer = make([]byte, 4096)
    var m msg.Message
    //var data string
    var timeout time.Duration
    timeout = time.Duration(0)

    for {
        select {
            case timeout = <-timeoutCh: {}
            default: {
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
       
    }

}



func processMyDataMsg(MyID int, 
                      LeftID int,
                      RightID int,
                      m msg.Message) msg.Message {
    m_data := msg.FromJsonDataMsg([]byte(m.Data_))
    switch m_data.Type_ {
        case "conf": {
            //got confirmation, refresh token
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
            // need to send confirmation
            confirmation_m := msg.DataMessage{Type_: "conf", Dst_: m_data.Src_, Src_: MyID, Data_: ""}
            m = msg.Message{Type_: "token", Dst_: m_data.Src_, Data_: string(confirmation_m.ToJsonDataMsg())}
        }
    }
    return m
}


func processEmptyTokenAndCheckMaintance(MyID int,
                                        LeftID int,
                                        RightID int,
                                        m msg.Message,
                                        taskCh chan msg.Message,
                                        needDrop *bool) msg.Message {
    select {
        case tmp := <-taskCh:
            //we have unfulfilled maintance task
            switch tmp.Type_ {
                case "send":
                    m_data := msg.DataMessage{Type_: "send", Dst_: tmp.Dst_, Src_: MyID, Data_: tmp.Data_}
                    m = msg.Message{Type_: "token", Dst_: tmp.Dst_, Data_: string(m_data.ToJsonDataMsg())}
                case "drop":
                    *needDrop = true
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
    return m
}


func launchElection(MyID int, LeftID int, RightID int,
                    MyConn *net.UDPConn, LeftAddr *net.UDPAddr, RightAddr *net.UDPAddr,
                    electDoubled []bool, msgReturned *bool, generated *bool, noToken *bool,
                    delay time.Duration) {
    // timeout, initialize election
    fmt.Println("node", 
                MyID, 
                ": received timeout, launching election")
    

    //time.Sleep(time.Millisecond * delay)
    *noToken = true

    elect_m := msg.DataMessage{Type_: "elect", Dst_: RightID, Src_: MyID, Data_: strconv.Itoa(MyID)}
    m := msg.Message{Type_: "elect", Dst_: MyID, Data_: string(elect_m.ToJsonDataMsg())}

    buffer := m.ToJsonMsg()
    time.Sleep(delay)
    _, err := MyConn.WriteToUDP(buffer, RightAddr)
    msg.CheckError(err)

    elect_m = msg.DataMessage{Type_: "elect", Dst_: LeftID, Src_: MyID, Data_: strconv.Itoa(MyID)}
    m = msg.Message{Type_: "elect", Dst_: MyID, Data_: string(elect_m.ToJsonDataMsg())}

    buffer = m.ToJsonMsg()
    time.Sleep(delay)
    _, err = MyConn.WriteToUDP(buffer, LeftAddr)
    msg.CheckError(err)

    for i := range electDoubled {
        electDoubled[i] = false
    }
    electDoubled[MyID] = true
    *msgReturned = false
    *generated = false
}


func processElection(MyID int, LeftID int, RightID int, NProc int,
                     MyConn *net.UDPConn, LeftAddr *net.UDPAddr, RightAddr *net.UDPAddr,
                     m msg.Message,
                     electDoubled []bool,
                     noToken *bool, generated *bool, msgReturned *bool, 
                     delay time.Duration) {

    var err error
    if m.Dst_ == MyID {
        if !*msgReturned {
            m_tmp := msg.FromJsonDataMsg(([]byte)(m.Data_))
            *msgReturned = true
            for i := 0; i < NProc; i++ {
                if !isProcInList(m_tmp.Data_, i) {
                    *msgReturned = false
                    break
                }
            }
        }
        if *msgReturned {
            // election token came back
            m = msg.Message{Type_: "electfin", Dst_: MyID, Data_: m.Data_}

            buffer := m.ToJsonMsg()
            time.Sleep(delay)
            _, err = MyConn.WriteToUDP(buffer, RightAddr)
            msg.CheckError(err)
        }
    } else {
        // foreign elect token, pass further updated token
        fmt.Println("node", MyID, ": received election token", "with data:", m)
        if !*generated {
            *noToken = true
        }

        if !electDoubled[m.Dst_] {
            // duplicate elect token
             //fmt.Println()
            m_tmp := msg.FromJsonDataMsg([]byte(m.Data_))
            m_tmp.Src_ = MyID
            m_tmp.Dst_ = RightID
            updateProcList(&m_tmp.Data_, MyID)
            m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

            buffer := m.ToJsonMsg()
            time.Sleep(delay)
            _, err = MyConn.WriteToUDP(buffer, RightAddr)
            msg.CheckError(err)

            m_tmp.Dst_ = LeftID
            m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

            buffer = m.ToJsonMsg()
            time.Sleep(delay)
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

                buffer := m.ToJsonMsg()
                time.Sleep(delay)
                _, err = MyConn.WriteToUDP(buffer, RightAddr)
                msg.CheckError(err)
            } else {
                m_tmp.Src_ = MyID
                m_tmp.Dst_ = LeftID
                updateProcList(&m_tmp.Data_, MyID)
                m = msg.Message{Type_: "elect", Dst_: m.Dst_, Data_: string(m_tmp.ToJsonDataMsg())}

                buffer := m.ToJsonMsg()
                time.Sleep(delay)
                _, err = MyConn.WriteToUDP(buffer, LeftAddr)
                msg.CheckError(err)
            }
        }
    }
}



func processElectionFinish(MyID int, 
                           MyConn *net.UDPConn, RightAddr *net.UDPAddr,
                           m msg.Message, 
                           msgReturned *bool, generated *bool, noToken *bool,
                           delay time.Duration) {
    var err error
    var lastElectMsg msg.Message
    if *noToken {
        *msgReturned = true
        fmt.Println("node", 
                    MyID, 
                    ": received election token", 
                    "with data:",
                    m)
        lastElectMsg = m
        if m.Dst_ != MyID {
            // pass electfin token further
            buffer := m.ToJsonMsg()
            time.Sleep(delay)
            _, err = MyConn.WriteToUDP(buffer, RightAddr)
            msg.CheckError(err)
        }
        max := maxProcID(msg.FromJsonDataMsg(([]byte)(lastElectMsg.Data_)).Data_)
        if max == MyID && !*generated {
            fmt.Println("node", 
                        MyID, 
                        ": generated token")
            // generate new token
            m_tmp := msg.DataMessage{Type_: "empty", Dst_: -1, Data_: ""}
            m = msg.Message{Type_: "token", Dst_: -1, Data_: string(m_tmp.ToJsonDataMsg())}
            buffer := m.ToJsonMsg()

            time.Sleep(delay)
            _,err := MyConn.WriteToUDP(buffer, RightAddr)
            msg.CheckError(err)
            *generated = true
        }
        *noToken = false
    }
}





func proc(MyID int, 
          NProc int,
          PortArr []int,
          MaintArr []int, 
          quitCh chan struct{},
          delay time.Duration) {

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
    timeoutCh := make(chan time.Duration, 1)
    timeoutMaintCh := make(chan time.Duration, 1)

    timeoutCh <- delay * time.Duration(NProc) * 2
    timeoutMaintCh <- time.Second * 0

    go ManageConn(MyConn, dataCh, timeoutCh)
    go ManageConn(MyMaintConn, maintCh, timeoutMaintCh)

    buffer := make([]byte, 4096)
    var m msg.Message
    var needDrop bool
    needDrop = false

    var noToken bool
    noToken = false
    var generated bool
    generated = false
    electDoubled := make([]bool, NProc)
    for i := range electDoubled {
        electDoubled[i] = false
    }
 
    var msgReturned bool
    msgReturned = false

    for {
        select {
            case m = <- dataCh: {}
            case m = <- maintCh: {}
        }
        if m.Type_ == "token" {
            // Ordinary message
            if !noToken { // discard old token if nesessary
                if m.Dst_ == MyID {

                    m = processMyDataMsg(MyID, LeftID, RightID, m)

                } else if m.Dst_ == -1 {

                    m = processEmptyTokenAndCheckMaintance(MyID, LeftID, RightID, m, taskCh, &needDrop)

                } else {
                    //pass non empty token further
                    fmt.Println("node", MyID, ": received token from node", LeftID, ", sending token to node", RightID)
                }

                if !needDrop { 
                    buffer = m.ToJsonMsg()

                    time.Sleep(delay)
                    _, err = MyConn.WriteToUDP(buffer, RightAddr)
                    msg.CheckError(err)
                }
                needDrop = false

            } else {
                fmt.Println(MyID, " discarded token")
            }
        } else if m.Type_ == "timeout" {

            launchElection(MyID, LeftID, RightID, 
                           MyConn, LeftAddr, RightAddr, 
                           electDoubled, &msgReturned, &generated, &noToken,
                           delay)

        } else if m.Type_ == "elect" {
            processElection(MyID, LeftID, RightID, NProc,
                            MyConn, LeftAddr, RightAddr, 
                            m,
                            electDoubled,
                            &noToken, &generated, &msgReturned,
                            delay)

        } else if m.Type_ == "electfin" {

            processElectionFinish(MyID, 
                                  MyConn, RightAddr,
                                  m, 
                                  &msgReturned, &generated, &noToken,
                                  delay)
        } else {
            //Maintance message
            fmt.Println("node", 
                        MyID, 
                        ": received service message:",
                        string(m.ToJsonMsg()))
            switch m.Type_{
                case "send":
                    taskCh <- m
                case "drop":
                    taskCh <- m
                default: 
                    fmt.Println("WTF")
            }
        }

    }

    <-quitCh
    quitCh <- struct{}{}
}
