package main
 
import (
    "time"
)



func main() { 
    quitCh := make(chan struct{})
    NProc := 3
    BasePort := 30000
    BaseMaintPort := 40000


    for i := 0; i < NProc; i++ {
    	MyID := i
        MyPort := BasePort+i
        MyMaintPort := BaseMaintPort+i
        LeftID := i+1
        RightID := i-1
        LeftPort  := BasePort+((i-1)%NProc+NProc)%NProc
        LeftMaintPort  := BaseMaintPort+((i-1)%NProc+NProc)%NProc
        RightPort  := BasePort+((i+1)%NProc+NProc)%NProc
        RightMaintPort  := BaseMaintPort+((i+1)%NProc+NProc)%NProc
        go proc(MyID, MyPort, MyMaintPort, LeftID, LeftPort, LeftMaintPort, RightID, RightPort, RightMaintPort, quitCh)
    }

    <-quitCh
    time.Sleep(time.Millisecond * 100)

}