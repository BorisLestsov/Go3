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
    TokenID          int        `json:"id"`         //целое число, id токена
    Msgtype          int        `json:"msgtype"`    //тип сообщения
    FromID           int        `json:"from"`       //id узла, отправившего сообщение
    ToID             int        `json:"to"`         //id узла, которому предназначено сообщение
    Data             string     `json:"data"`       //строка, содрежащая данные
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
