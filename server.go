package main
 
import (
    "fmt"
    "net"
    msg "./message"
)


func main() {

    



    /* Lets prepare a address at any address at port 10001*/   
    ServerAddr, err := net.ResolveUDPAddr("udp",":10001")
    msg.CheckError(err)
 
    /* Now listen at selected port */
    ServerConn, err := net.ListenUDP("udp", ServerAddr)
    msg.CheckError(err)
    defer ServerConn.Close()
 
    buf := make([]byte, 1024)
 
    for {
        n,addr,err := ServerConn.ReadFromUDP(buf)
        fmt.Println("Received ",string(buf[0:n]), " from ",addr)

        m := msg.FromJson(buf[0:n])
        fmt.Println(m)
 
        if err != nil {
            fmt.Println("Error: ",err)
        } 
    }
}