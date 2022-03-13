package api

import (
	"fmt"
	"strings"

	spb "github.com/openconfig/gribi/v1/proto/service"
	"google.golang.org/protobuf/proto"
)

func NewGetRequest(opts ...GRIBIOption) (*spb.GetRequest, error) {
	m := new(spb.GetRequest)
	err := apply(m, opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func NSAll() func(m proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.GetRequest:
			msg.NetworkInstance = &spb.GetRequest_All{}
		case *spb.FlushRequest:
			msg.NetworkInstance = &spb.FlushRequest_All{}
		default:
			return fmt.Errorf("option NSAll: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func AFTType(typ string) func(m proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if typ == "" {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.GetRequest:
			switch strings.ToUpper(typ) {
			case "ALL":
				msg.Aft = spb.AFTType_ALL
			case "IPV4":
				msg.Aft = spb.AFTType_IPV4
			case "IPV6":
				msg.Aft = spb.AFTType_IPV6
			case "MPLS":
				msg.Aft = spb.AFTType_MPLS
			case "NEXTHOP", "NH":
				msg.Aft = spb.AFTType_NEXTHOP
			case "NEXTHOP_GROUP", "NEXTHOP-GROUP", "NHG":
				msg.Aft = spb.AFTType_NEXTHOP_GROUP
			case "MAC":
				msg.Aft = spb.AFTType_MAC
			case "POLICY_FORWARDING", "POLICY-FORWARDING", "PF":
				msg.Aft = spb.AFTType_POLICY_FORWARDING
			default:
				return fmt.Errorf("option AFTType: %w: %v", ErrInvalidValue, typ)
			}
		default:
			return fmt.Errorf("option AFTType: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}
