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
    Id          int        `json:"id"`         //целое число, id сообщения
    Msgtype     int        `json:"msgtype"`    //тип сообщения (multicast, notication)
    Sender      int        `json:"sender"`     //номер узла, отправляющего сообщение
    Origin      int        `json:"origin"`     //номер узла, отправившего исходное сообщение
    Data        string     `json:"data"`       //строка, содрежащая данные
}


func (msg Message) ToJson() []byte {
    buf, err := json.Marshal(msg)
    CheckError(err)
    return buf
}

func FromJson(buffer []byte) Message {
	var msg Message
    err := json.Unmarshal(buffer, &msg)
    CheckError(err)
    return msg
}
