package main
 
import (
    "fmt"
    "net"
    "time"
    "io/ioutil"
    msg "./message"
)


func main() {

    ServerAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:10001")
    msg.CheckError(err)
 
    LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
    msg.CheckError(err)
 
    Conn, err := net.DialUDP("udp", LocalAddr, ServerAddr)
    msg.CheckError(err)
 
    tmp, err := ioutil.ReadFile("./data/data.txt")
    msg.CheckError(err)

    buf := string(tmp)
    fmt.Println("Data length: ", len(buf))

    defer Conn.Close()
    i := 0
    for {

        m := msg.Message{Id: i, Msgtype: 1, Sender: 2, Origin: 3, Data: buf}
        fmt.Println(m)

        buffer := m.ToJson()
        fmt.Println(string(buffer))

        i++

        _,err := Conn.Write(buffer)
        if err != nil {
            panic(err)
        }
        time.Sleep(time.Second * 1)
    }
}