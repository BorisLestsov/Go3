package message

import (
    "encoding/json"
)


/* A Simple function to verify error */
func CheckError(err error) {
    if err  != nil {
        panic(err)
    }
}


type Message struct {
    Type_         string     `json:"type"`         //тип сообщения
    Dst_          int        `json:"dst"`          //получатель
    Data_         string     `json:"data"`         //строка, содрежащая данные
}

type DataMessage struct {
    Type_         string     `json:"type"`         //тип сообщения
    Src_          int        `json:"src"`          //отправитель
    Dst_          int        `json:"dst"`          //получатель
    Data_         string     `json:"data"`         //строка, содрежащая данные
}


func (msg Message) ToJsonMsg() []byte {
    buf, err := json.Marshal(msg)
    CheckError(err)
    return buf
}

func FromJsonMsg(buffer []byte) Message {
	var msg Message
    err := json.Unmarshal(buffer, &msg)
    CheckError(err)
    return msg
}


func (msg DataMessage) ToJsonDataMsg() []byte {
    buf, err := json.Marshal(msg)
    CheckError(err)
    return buf
}

func FromJsonDataMsg(buffer []byte) DataMessage {
    var msg DataMessage
    err := json.Unmarshal(buffer, &msg)
    CheckError(err)
    return msg
}
