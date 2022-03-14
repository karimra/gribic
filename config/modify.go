package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type OperationConfig struct {
	ID              uint64 `yaml:"id,omitempty"`
	NetworkInstance string `yaml:"network-instance,omitempty"`
	Operation       string `yaml:"op,omitempty"`
	//
	IPv6 *ipv4v6Entry `yaml:"ipv6,omitempty"`
	IPv4 *ipv4v6Entry `yaml:"ipv4,omitempty"`
	NHG  *nhgEntry    `yaml:"nhg,omitempty"`
	NH   *nhEntry     `yaml:"nh,omitempty"`
	//
	ElectionID uint64 `yaml:"election-id,omitempty"`
}

type modifyInput struct {
	DefaultNetworkInstance string `yaml:"default-network-instance"`
	Params                 struct {
		Redundancy  string `yaml:"redundancy,omitempty"`
		Persistence string `yaml:"persistence,omitempty"`
		AckType     string `yaml:"ack-type,omitempty"`
	} `json:"params,omitempty"`
	Operations []*OperationConfig `yaml:"operations,omitempty"`
}

type ipv4v6Entry struct {
	Type string `yaml:"type,omitempty"`
	// ipv4v6
	Prefix             string `yaml:"prefix,omitempty"`
	NHG                uint64 `yaml:"nhg,omitempty"`
	NHGNetworkInstance string `yaml:"nhg-network-instance,omitempty"`
	DecapsulateHeader  string `yaml:"decapsulate-header,omitempty"`
	EntryMetadata      string `yaml:"entry-metadata,omitempty"`
}

type nhgEntry struct {
	Type string `yaml:"type,omitempty"`
	// nhg
	ID        uint64  `yaml:"id,omitempty"`
	BackupNHG *uint64 `yaml:"backup-nhg,omitempty"`
	Color     *uint64 `yaml:"color,omitempty"`
	NextHop   []struct {
		Index  uint64 `yaml:"index,omitempty"`
		Weight uint64 `yaml:"weight,omitempty"`
	} `yaml:"next-hop,omitempty"`
	ProgrammedID *uint64 `yaml:"programmed-id,omitempty"`
}

type nhEntry struct {
	Type string `yaml:"type,omitempty"`
	// nh
	Index                uint64              `yaml:"index,omitempty"`
	DecapsulateHeader    string              `yaml:"decapsulate-header,omitempty"`
	EncapsulateHeader    string              `yaml:"encapsulate-header,omitempty"`
	IPAddress            string              `yaml:"ip-address,omitempty"`
	InterfaceReference   *interfaceReference `yaml:"interface-reference,omitempty"`
	IPinIP               *ipinip             `yaml:"ip-in-ip,omitempty"`
	MAC                  string              `yaml:"mac,omitempty"`
	NetworkInstance      string              `yaml:"network-instance,omitempty"`
	ProgrammedIndex      *uint64             `yaml:"programmed-index,omitempty"`
	PushedMPLSLabelStack []struct {
		Type  string `yaml:"type,omitempty"`
		Label uint   `yaml:"label,omitempty"`
	} `yaml:"pushed-mpls-label-stack,omitempty"`
}

func ReadModifyFile(file string) (*modifyInput, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	mi := new(modifyInput)
	err = yaml.Unmarshal(b, mi)
	if err != nil {
		return nil, err
	}
	for i, op := range mi.Operations {
		if op.NetworkInstance == "" {
			mi.Operations[i].NetworkInstance = mi.DefaultNetworkInstance
		}
		mi.Operations[i].ID = uint64(i) + 1
	}
	return mi, nil
}

type interfaceReference struct {
	Interface    string  `yaml:"interface,omitempty"`
	Subinterface *uint64 `yaml:"subinterface,omitempty"`
}

type ipinip struct {
	DSTIP string `yaml:"dst-ip,omitempty"`
	SRCIP string `yaml:"src-ip,omitempty"`
}
