package main

import (
	"errors"
	"fmt"
)

type Controller interface {
	HandleBindRequest(msgId int64, br *BindRequest) (Message, error)
	HandleAddRequest(msgId int64, ar *AddRequest) (Message, error)
}

func HandleMessage(c Controller, m Message) (Message, error) {
	switch protoOp := m.ProtocolOp.(type) {
	case *BindRequest:
		return c.HandleBindRequest(m.MessageId, protoOp)
	case *AddRequest:
		return c.HandleAddRequest(m.MessageId, protoOp)
	}

	return Message{}, fmt.Errorf("Unknown/unimplemented message type")
}

type ControllerImpl struct {
	entryService EntryService
}

func NewController(entryService EntryService) Controller {
	return &ControllerImpl{entryService}
}

func (c *ControllerImpl) HandleBindRequest(msgId int64, br *BindRequest) (Message, error) {
	bindRes := BindResponse{
		ResultCode:        Success,
		MatchedDN:         br.name,
		DiagnosticMessage: "Passwords not implemented yet - you seem trustworthy though",
	}

	m := Message{
		MessageId:  msgId,
		ProtocolOp: &bindRes,
	}

	return m, nil
}

func (c *ControllerImpl) HandleAddRequest(msgId int64, ar *AddRequest) (Message, error) {
	_, err := c.entryService.AddEntry(ar)

	if err != nil {
		var ldapErr LDAPError
		if errors.As(err, &ldapErr) {
			resp := AddResponse(ldapErr.Result())
			return Message{msgId, &resp}, nil
		}

		return Message{}, err
	}

	resp := AddResponse{
		ResultCode: Success,
		MatchedDN:  ar.Entry,
	}

	return Message{msgId, &resp}, nil
}
