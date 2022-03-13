package api

import (
	spb "github.com/openconfig/gribi/v1/proto/service"
	"google.golang.org/protobuf/proto"
)

func NewModifyRequest(opts ...GRIBIOption) (*spb.ModifyRequest, error) {
	m := new(spb.ModifyRequest)
	err := apply(m, opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func RedundancyAllPrimary() func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			if msg.Params == nil {
				msg.Params = new(spb.SessionParameters)
			}
			msg.Params.Redundancy = spb.SessionParameters_ALL_PRIMARY
		}
		return nil
	}
}

func RedundancySinglePrimary() func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			if msg.Params == nil {
				msg.Params = new(spb.SessionParameters)
			}
			msg.Params.Redundancy = spb.SessionParameters_SINGLE_PRIMARY
		}
		return nil
	}
}

func PersistenceDelete() func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			if msg.Params == nil {
				msg.Params = new(spb.SessionParameters)
			}
			msg.Params.Persistence = spb.SessionParameters_DELETE
		}
		return nil
	}
}

func PersistencePreserve() func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			if msg.Params == nil {
				msg.Params = new(spb.SessionParameters)
			}
			msg.Params.Persistence = spb.SessionParameters_PRESERVE
		}
		return nil
	}
}

func AckTypeRib() func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			if msg.Params == nil {
				msg.Params = new(spb.SessionParameters)
			}
			msg.Params.AckType = spb.SessionParameters_RIB_ACK
		}
		return nil
	}
}

func AckTypeRibFib() func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			if msg.Params == nil {
				msg.Params = new(spb.SessionParameters)
			}
			msg.Params.AckType = spb.SessionParameters_RIB_AND_FIB_ACK
		}
		return nil
	}
}

func AFTOperation(opts ...GRIBIOption) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.ModifyRequest:
			aftOper, err := NewAFTOperation(opts...)
			if err != nil {
				return err
			}
			if len(msg.Operation) == 0 {
				msg.Operation = make([]*spb.AFTOperation, 0)
			}
			msg.Operation = append(msg.Operation, aftOper)
		}
		return nil
	}
}
