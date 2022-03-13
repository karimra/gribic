package api

import (
	"fmt"

	spb "github.com/openconfig/gribi/v1/proto/service"
	"google.golang.org/protobuf/proto"
)

func NewFlushRequest(opts ...GRIBIOption) (*spb.FlushRequest, error) {
	m := new(spb.FlushRequest)
	err := apply(m, opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func Override() func(m proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.FlushRequest:
			msg.Election = &spb.FlushRequest_Override{}
		default:
			return fmt.Errorf("option Override: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}
