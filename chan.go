package main
 
import (
    "time"
)



func main() { 
    quitCh := make(chan struct{})
    NProc := 5
    BasePort := 30000
    BaseMaintPort := 40000


    for i := 0; i < NProc; i++ {
    	MyID := i
        MyPort := BasePort+i
        MyMaintPort := BaseMaintPort+i
        LeftID := ((i-1)%NProc+NProc)%NProc
        RightID := ((i+1)%NProc+NProc)%NProc
        LeftPort  := BasePort+((i-1)%NProc+NProc)%NProc
        RightPort  := BasePort+((i+1)%NProc+NProc)%NProc
        go proc(MyID, MyPort, MyMaintPort, LeftID, LeftPort, RightID, RightPort, quitCh)
    }

    for i := 0; i < NProc; i++ {
	    <-quitCh
	}
    time.Sleep(time.Millisecond * 100)

}