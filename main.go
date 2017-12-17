package main
 
import (
    "time"
    "os"
    "strconv"
)



func main() { 
    quitCh := make(chan struct{})
    NProc, _ := strconv.Atoi(os.Args[1])
    BasePort, _ := strconv.Atoi(os.Args[2])
    BaseMaintPort, _ := strconv.Atoi(os.Args[3])
    d, _ := strconv.Atoi(os.Args[4])
    delay := time.Millisecond*time.Duration(d)

    PortArr  := make([]int, NProc)
    MaintArr := make([]int, NProc)
    for i := 0; i < NProc; i++ {
    	PortArr[i] = BasePort + i
    	MaintArr[i] = BaseMaintPort + i
    }

    for i := 0; i < NProc; i++ {
        go proc(i, NProc, PortArr, MaintArr, quitCh, delay)
    }

    for i := 0; i < NProc; i++ {
	    <-quitCh
	}
    time.Sleep(time.Millisecond * 100)

}