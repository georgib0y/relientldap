package main

import "fmt"

type Controller interface {
	HandleBindRequest(msgId int64, br *BindRequest) (Message, error)
}

func HandleMessage(c Controller, m Message) (Message, error) {
	switch protoOp := m.ProtocolOp.(type) {
	case *BindRequest:
		return c.HandleBindRequest(m.MessageId, protoOp)
	}

	return Message{}, fmt.Errorf("Unknown/unimplemented message type")
}

type ControllerImpl struct{}

func (c *ControllerImpl) HandleBindRequest(msgId int64, br *BindRequest) (Message, error) {
	bindRes := BindResponse{Result{
		ResultCode:        ResultSuccess,
		MatchedDN:         br.name,
		DiagnosticMessage: "Passwords not implemented yet - you seem trustworthy though",
	}}

	m := Message{
		MessageId:  msgId,
		ProtocolOp: &bindRes,
	}

	return m, nil
}
