package api

import (
	"fmt"
	"strings"

	gribi_aft "github.com/openconfig/gribi/v1/proto/gribi_aft"
	"github.com/openconfig/gribi/v1/proto/gribi_aft/enums"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/openconfig/ygot/proto/ywrapper"
	"google.golang.org/protobuf/proto"
)

func NewAFTOperation(opts ...GRIBIOption) (*spb.AFTOperation, error) {
	m := new(spb.AFTOperation)
	err := apply(m, opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// AFTOperation ID or NextHopGroup ID
func ID(id uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			msg.Id = id
		case *gribi_aft.Afts_NextHopGroupKey:
			msg.Id = id
		default:
			return fmt.Errorf("option ID: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

// AFTOperation Network Instance, or
// NextHop Entry Network Instance, or
// IPv4 Entry Network Instance.
func NetworkInstance(ns string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if ns == "" {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.GetRequest:
			msg.NetworkInstance = &spb.GetRequest_Name{Name: ns}
		case *spb.FlushRequest:
			msg.NetworkInstance = &spb.FlushRequest_Name{Name: ns}
		case *spb.AFTOperation:
			msg.NetworkInstance = ns
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			msg.NextHop.NetworkInstance = &ywrapper.StringValue{Value: ns}
		case *gribi_aft.Afts_Ipv4EntryKey:
			if msg.Ipv4Entry == nil {
				msg.Ipv4Entry = new(gribi_aft.Afts_Ipv4Entry)
			}
			msg.Ipv4Entry.NextHopGroupNetworkInstance = &ywrapper.StringValue{Value: ns}
		case *gribi_aft.Afts_Ipv6EntryKey:
			if msg.Ipv6Entry == nil {
				msg.Ipv6Entry = new(gribi_aft.Afts_Ipv6Entry)
			}
			msg.Ipv6Entry.NextHopGroupNetworkInstance = &ywrapper.StringValue{Value: ns}
		default:
			return fmt.Errorf("option NetworkInstance: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

// AFTOperation Network Instance
func Op(op string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			switch strings.ToUpper(op) {
			case "ADD":
				msg.Op = spb.AFTOperation_ADD
			case "REPLACE":
				msg.Op = spb.AFTOperation_REPLACE
			case "DELETE":
				msg.Op = spb.AFTOperation_DELETE
			default:
				return fmt.Errorf("option Op: %w: %T", ErrInvalidValue, msg)
			}
		default:
			return fmt.Errorf("option Op: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func OpAdd() func(proto.Message) error {
	return Op("ADD")
}

func OpReplace() func(proto.Message) error {
	return Op("REPLACE")
}

func OpDelete() func(proto.Message) error {
	return Op("DELETE")
}

func ElectionID(id *spb.Uint128) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if id == nil {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			msg.ElectionId = id
		case *spb.FlushRequest:
			msg.Election = &spb.FlushRequest_Id{Id: id}
		case *spb.ModifyRequest:
			msg.ElectionId = id
		default:
			return fmt.Errorf("option ElectionID: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

// Next Hop Options
func NHEntry(opts ...GRIBIOption) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			nh := new(gribi_aft.Afts_NextHopKey)
			err := apply(nh, opts...)
			if err != nil {
				return err
			}
			msg.Entry = &spb.AFTOperation_NextHop{
				NextHop: nh,
			}
			return nil
		default:
			return fmt.Errorf("option EntryNH: %w: %T", ErrInvalidMsgType, msg)
		}
	}
}

func Index(index uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			msg.Index = index
			return nil
		default:
			return fmt.Errorf("option Index: %w: %T", ErrInvalidMsgType, msg)
		}
	}
}

func IPAddress(ipAddr string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if ipAddr == "" {
				return nil
			}
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			msg.NextHop.IpAddress = &ywrapper.StringValue{Value: ipAddr}
		default:
			return fmt.Errorf("option IPAddress: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func DecapsulateHeader(typ string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if typ == "" {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			switch strings.ToUpper(typ) {
			case "GRE":
				msg.NextHop.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_GRE
			case "IPV4":
				msg.NextHop.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV4
			case "IPV6":
				msg.NextHop.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV6
			case "MPLS":
				msg.NextHop.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_MPLS
			default:
				return fmt.Errorf("option DecapsulateHeader: %w: %T", ErrInvalidValue, msg)
			}
		case *gribi_aft.Afts_Ipv4EntryKey:
			if msg.Ipv4Entry == nil {
				msg.Ipv4Entry = new(gribi_aft.Afts_Ipv4Entry)
			}
			switch strings.ToUpper(typ) {
			case "GRE":
				msg.Ipv4Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_GRE
			case "IPV4":
				msg.Ipv4Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV4
			case "IPV6":
				msg.Ipv4Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV6
			case "MPLS":
				msg.Ipv4Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_MPLS
			default:
				return fmt.Errorf("option DecapsulateHeader: %w: %T", ErrInvalidValue, msg)
			}
		case *gribi_aft.Afts_Ipv6EntryKey:
			if msg.Ipv6Entry == nil {
				msg.Ipv6Entry = new(gribi_aft.Afts_Ipv6Entry)
			}
			switch strings.ToUpper(typ) {
			case "GRE":
				msg.Ipv6Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_GRE
			case "IPV4":
				msg.Ipv6Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV4
			case "IPV6":
				msg.Ipv6Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV6
			case "MPLS":
				msg.Ipv6Entry.DecapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_MPLS
			default:
				return fmt.Errorf("option DecapsulateHeader: %w: %T", ErrInvalidValue, msg)
			}
		default:
			return fmt.Errorf("option DecapsulateHeader: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func DecapsulateHeaderGRE() func(proto.Message) error {
	return DecapsulateHeader("GRE")
}

func DecapsulateHeaderIPv4() func(proto.Message) error {
	return DecapsulateHeader("IPV4")
}

func DecapsulateHeaderIPv6() func(proto.Message) error {
	return DecapsulateHeader("IPV6")
}

func DecapsulateHeaderMPLS() func(proto.Message) error {
	return DecapsulateHeader("MPLS")
}

func EncapsulateHeader(typ string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if typ == "" {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			switch strings.ToUpper(typ) {
			case "GRE":
				msg.NextHop.EncapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_GRE
			case "IPV4":
				msg.NextHop.EncapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV4
			case "IPV6":
				msg.NextHop.EncapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_IPV6
			case "MPLS":
				msg.NextHop.EncapsulateHeader = enums.OpenconfigAftTypesEncapsulationHeaderType_OPENCONFIGAFTTYPESENCAPSULATIONHEADERTYPE_MPLS
			default:
				return fmt.Errorf("option EncapsulateHeader: %w: %T", ErrInvalidValue, msg)
			}
		default:
			return fmt.Errorf("option EncapsulateHeader: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func EncapsulateHeaderGRE() func(proto.Message) error {
	return EncapsulateHeader("GRE")
}

func EncapsulateHeaderIPv4() func(proto.Message) error {
	return EncapsulateHeader("IPV4")
}

func EncapsulateHeaderIPv6() func(proto.Message) error {
	return EncapsulateHeader("IPV6")
}

func EncapsulateHeaderMPLS() func(proto.Message) error {
	return EncapsulateHeader("MPLS")
}

func Interface(iface string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if iface == "" {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			if msg.NextHop.InterfaceRef == nil {
				msg.NextHop.InterfaceRef = new(gribi_aft.Afts_NextHop_InterfaceRef)
			}
			msg.NextHop.InterfaceRef.Interface = &ywrapper.StringValue{Value: iface}
		default:
			return fmt.Errorf("option Interface: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func SubInterface(subIface uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			if msg.NextHop.InterfaceRef == nil {
				msg.NextHop.InterfaceRef = new(gribi_aft.Afts_NextHop_InterfaceRef)
			}
			msg.NextHop.InterfaceRef.Subinterface = &ywrapper.UintValue{Value: subIface}
		default:
			return fmt.Errorf("option SubInterface: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func IPinIP(src, dst string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			if msg.NextHop.IpInIp == nil {
				msg.NextHop.IpInIp = new(gribi_aft.Afts_NextHop_IpInIp)
			}
			if src != "" {
				msg.NextHop.IpInIp.SrcIp = &ywrapper.StringValue{Value: src}
			}
			if dst != "" {
				msg.NextHop.IpInIp.DstIp = &ywrapper.StringValue{Value: dst}
			}
		default:
			return fmt.Errorf("option IPinIP: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func MAC(mac string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if mac == "" {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			msg.NextHop.MacAddress = &ywrapper.StringValue{Value: mac}
		default:
			return fmt.Errorf("option MAC: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func PushedMplsLabelStack(typ string, label uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopKey:
			if msg.NextHop == nil {
				msg.NextHop = new(gribi_aft.Afts_NextHop)
			}
			if msg.NextHop.PushedMplsLabelStack == nil {
				msg.NextHop.PushedMplsLabelStack = make([]*gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion, 0)
			}
			typ = strings.ToUpper(typ)
			typ = strings.ReplaceAll(typ, "-", "_")
			switch typ {
			case "IPV4_EXPLICIT_NULL":
				msg.NextHop.PushedMplsLabelStack = append(msg.NextHop.PushedMplsLabelStack,
					&gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion{
						PushedMplsLabelStackOpenconfigmplstypesmplslabelenum: enums.OpenconfigMplsTypesMplsLabelEnum_OPENCONFIGMPLSTYPESMPLSLABELENUM_IPV4_EXPLICIT_NULL,
						PushedMplsLabelStackUint64:                           label,
					})
			case "ROUTER_ALERT":
				msg.NextHop.PushedMplsLabelStack = append(msg.NextHop.PushedMplsLabelStack,
					&gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion{
						PushedMplsLabelStackOpenconfigmplstypesmplslabelenum: enums.OpenconfigMplsTypesMplsLabelEnum_OPENCONFIGMPLSTYPESMPLSLABELENUM_ROUTER_ALERT,
						PushedMplsLabelStackUint64:                           label,
					})
			case "IPV6_EXPLICIT_NULL":
				msg.NextHop.PushedMplsLabelStack = append(msg.NextHop.PushedMplsLabelStack,
					&gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion{
						PushedMplsLabelStackOpenconfigmplstypesmplslabelenum: enums.OpenconfigMplsTypesMplsLabelEnum_OPENCONFIGMPLSTYPESMPLSLABELENUM_IPV6_EXPLICIT_NULL,
						PushedMplsLabelStackUint64:                           label,
					})
			case "IMPLICIT_NULL":
				msg.NextHop.PushedMplsLabelStack = append(msg.NextHop.PushedMplsLabelStack,
					&gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion{
						PushedMplsLabelStackOpenconfigmplstypesmplslabelenum: enums.OpenconfigMplsTypesMplsLabelEnum_OPENCONFIGMPLSTYPESMPLSLABELENUM_IMPLICIT_NULL,
						PushedMplsLabelStackUint64:                           label,
					})
			case "ENTROPY_LABEL_INDICATOR":
				msg.NextHop.PushedMplsLabelStack = append(msg.NextHop.PushedMplsLabelStack,
					&gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion{
						PushedMplsLabelStackOpenconfigmplstypesmplslabelenum: enums.OpenconfigMplsTypesMplsLabelEnum_OPENCONFIGMPLSTYPESMPLSLABELENUM_ENTROPY_LABEL_INDICATOR,
						PushedMplsLabelStackUint64:                           label,
					})
			case "NO_LABEL":
				msg.NextHop.PushedMplsLabelStack = append(msg.NextHop.PushedMplsLabelStack,
					&gribi_aft.Afts_NextHop_PushedMplsLabelStackUnion{
						PushedMplsLabelStackOpenconfigmplstypesmplslabelenum: enums.OpenconfigMplsTypesMplsLabelEnum_OPENCONFIGMPLSTYPESMPLSLABELENUM_NO_LABEL,
						// PushedMplsLabelStackUint64:                           label,
					})
			default:
				return fmt.Errorf("option PushedMplsLabelStack: %w: %T", ErrInvalidValue, msg)
			}
		default:
			return fmt.Errorf("option PushedMplsLabelStack: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func PushedMplsLabelStackIPv4(label uint64) func(proto.Message) error {
	return PushedMplsLabelStack("IPV4_EXPLICIT_NULL", label)
}

func PushedMplsLabelStackRouterAlert(label uint64) func(proto.Message) error {
	return PushedMplsLabelStack("ROUTER_ALERT", label)
}

func PushedMplsLabelStackRouterIPv6(label uint64) func(proto.Message) error {
	return PushedMplsLabelStack("IPV6_EXPLICIT_NULL", label)
}

func PushedMplsLabelStackRouterImplicit(label uint64) func(proto.Message) error {
	return PushedMplsLabelStack("IMPLICIT_NULL", label)
}

func PushedMplsLabelStackRouterEntropy(label uint64) func(proto.Message) error {
	return PushedMplsLabelStack("ENTROPY_LABEL_INDICATOR", label)
}

func PushedMplsLabelStackRouterNoLabel() func(proto.Message) error {
	return PushedMplsLabelStack("NO_LABEL", 0)
}

// Next Hop Group Options
func NHGEntry(opts ...GRIBIOption) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			nhg := new(gribi_aft.Afts_NextHopGroupKey)
			err := apply(nhg, opts...)
			if err != nil {
				return err
			}
			msg.Entry = &spb.AFTOperation_NextHopGroup{
				NextHopGroup: nhg,
			}
			return nil
		default:
			return fmt.Errorf("option EntryNHG: %w: %T", ErrInvalidMsgType, msg)
		}
	}
}

func BackupNextHopGroup(index uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopGroupKey:
			if msg.NextHopGroup == nil {
				msg.NextHopGroup = new(gribi_aft.Afts_NextHopGroup)
			}
			msg.NextHopGroup.BackupNextHopGroup = &ywrapper.UintValue{Value: index}
		default:
			return fmt.Errorf("option BackupNextHopGroup: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func Color(index uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopGroupKey:
			if msg.NextHopGroup == nil {
				msg.NextHopGroup = new(gribi_aft.Afts_NextHopGroup)
			}
			msg.NextHopGroup.Color = &ywrapper.UintValue{Value: index}
		default:
			return fmt.Errorf("option Color: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func NHGNextHop(index, weight uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_NextHopGroupKey:
			if msg.NextHopGroup == nil {
				msg.NextHopGroup = new(gribi_aft.Afts_NextHopGroup)
			}
			if len(msg.NextHopGroup.NextHop) == 0 {
				msg.NextHopGroup.NextHop = make([]*gribi_aft.Afts_NextHopGroup_NextHopKey, 0)
			}
			nhgnh := new(gribi_aft.Afts_NextHopGroup_NextHopKey)
			nhgnh.Index = index
			if weight > 0 {
				nhgnh.NextHop = &gribi_aft.Afts_NextHopGroup_NextHop{
					Weight: &ywrapper.UintValue{Value: weight},
				}
			}
			msg.NextHopGroup.NextHop = append(msg.NextHopGroup.NextHop, nhgnh)
		default:
			return fmt.Errorf("option NHGNextHop: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

// IPv4 Options
func IPv4Entry(opts ...GRIBIOption) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			ipv4 := new(gribi_aft.Afts_Ipv4EntryKey)
			err := apply(ipv4, opts...)
			if err != nil {
				return err
			}
			msg.Entry = &spb.AFTOperation_Ipv4{
				Ipv4: ipv4,
			}
			return nil
		default:
			return fmt.Errorf("option IPv4Entry: %w: %T", ErrInvalidMsgType, msg)
		}
	}
}

func Prefix(prefix string) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_Ipv4EntryKey:
			msg.Prefix = prefix
		case *gribi_aft.Afts_Ipv6EntryKey:
			msg.Prefix = prefix
		default:
			return fmt.Errorf("option Prefix: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func Metadata(md []byte) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		if len(md) == 0 {
			return nil
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_Ipv4EntryKey:
			if msg.Ipv4Entry == nil {
				msg.Ipv4Entry = new(gribi_aft.Afts_Ipv4Entry)
			}
			msg.Ipv4Entry.EntryMetadata = &ywrapper.BytesValue{Value: md}
		case *gribi_aft.Afts_Ipv6EntryKey:
			if msg.Ipv6Entry == nil {
				msg.Ipv6Entry = new(gribi_aft.Afts_Ipv6Entry)
			}
			msg.Ipv6Entry.EntryMetadata = &ywrapper.BytesValue{Value: md}
		default:
			return fmt.Errorf("option Metadata: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

func NHG(id uint64) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *gribi_aft.Afts_Ipv4EntryKey:
			if msg.Ipv4Entry == nil {
				msg.Ipv4Entry = new(gribi_aft.Afts_Ipv4Entry)
			}
			msg.Ipv4Entry.NextHopGroup = &ywrapper.UintValue{Value: id}
		case *gribi_aft.Afts_Ipv6EntryKey:
			if msg.Ipv6Entry == nil {
				msg.Ipv6Entry = new(gribi_aft.Afts_Ipv6Entry)
			}
			msg.Ipv6Entry.NextHopGroup = &ywrapper.UintValue{Value: id}
		default:
			return fmt.Errorf("option NHG: %w: %T", ErrInvalidMsgType, msg)
		}
		return nil
	}
}

// IPv6 Entry Options
func IPv6Entry(opts ...GRIBIOption) func(proto.Message) error {
	return func(msg proto.Message) error {
		if msg == nil {
			return ErrInvalidMsgType
		}
		switch msg := msg.ProtoReflect().Interface().(type) {
		case *spb.AFTOperation:
			ipv6 := new(gribi_aft.Afts_Ipv6EntryKey)
			err := apply(ipv6, opts...)
			if err != nil {
				return err
			}
			msg.Entry = &spb.AFTOperation_Ipv6{
				Ipv6: ipv6,
			}
			return nil
		default:
			return fmt.Errorf("option IPv6Entry: %w: %T", ErrInvalidMsgType, msg)
		}
	}
}
