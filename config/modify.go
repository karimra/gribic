package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/karimra/gnmic/utils"
	"github.com/karimra/gribic/api"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"gopkg.in/yaml.v2"
)

const (
	varFileSuffix = "_vars"
)

type OperationConfig struct {
	ID              uint64 `yaml:"id,omitempty" json:"id,omitempty"`
	NetworkInstance string `yaml:"network-instance,omitempty" json:"network-instance,omitempty"`
	Operation       string `yaml:"op,omitempty" json:"operation,omitempty"`
	//
	IPv6 *ipv4v6Entry `yaml:"ipv6,omitempty" json:"ipv6,omitempty"`
	IPv4 *ipv4v6Entry `yaml:"ipv4,omitempty" json:"ipv4,omitempty"`
	NHG  *nhgEntry    `yaml:"nhg,omitempty" json:"nhg,omitempty"`
	NH   *nhEntry     `yaml:"nh,omitempty" json:"nh,omitempty"`
	//
	ElectionID string `yaml:"election-id,omitempty" json:"election-id,omitempty"`
	//
	electionID *spb.Uint128
}

func (oc *OperationConfig) String() string {
	b, _ := json.MarshalIndent(oc, "", "  ")
	return string(b)
}

func (oc *OperationConfig) validate() error {
	if oc.IPv4 == nil && oc.IPv6 == nil && oc.NHG == nil && oc.NH == nil {
		return errors.New("missing entry")
	}
	if oc.IPv4 != nil {
		if oc.IPv6 != nil {
			return errors.New("both ipv4 and ipv6 entries are defined")
		}
		if oc.NHG != nil {
			return errors.New("both ipv4 and nhg entries are defined")
		}
		if oc.NH != nil {
			return errors.New("both ipv4 and nh entries are defined")
		}
		return nil
	}
	if oc.IPv6 != nil {
		if oc.NHG != nil {
			return errors.New("both ipv6 and nhg entries are defined")
		}
		if oc.NH != nil {
			return errors.New("both ipv6 and nh entries are defined")
		}
		return nil
	}
	if oc.NHG != nil {
		if oc.NH != nil {
			return errors.New("both nhg and nh entries are defined")
		}
		return nil
	}
	return nil
}

func (o *OperationConfig) calculateElectionID() error {
	if o.ElectionID == "" {
		return nil
	}
	var err error
	o.electionID, err = ParseUint128(o.ElectionID)
	return err
}

func (o *OperationConfig) CreateAftOper() (*spb.AFTOperation, error) {
	err := o.calculateElectionID()
	if err != nil {
		return nil, err
	}
	opts := []api.GRIBIOption{
		api.ID(o.ID),
		api.NetworkInstance(o.NetworkInstance),
		api.Op(o.Operation),
		api.ElectionID(o.electionID),
	}

	switch {
	case o.IPv6 != nil:
		// append IPv6Entry option
		opts = append(opts,
			api.IPv6Entry(
				api.Prefix(o.IPv6.Prefix),
				api.DecapsulateHeader(o.IPv6.DecapsulateHeader),
				api.Metadata([]byte(o.IPv6.EntryMetadata)),
				api.NHG(o.IPv6.NHG),
				api.NetworkInstance(o.IPv6.NHGNetworkInstance),
			),
		)
	case o.IPv4 != nil:
		// append IPv4Entry option
		opts = append(opts,
			api.IPv4Entry(
				api.Prefix(o.IPv4.Prefix),
				api.DecapsulateHeader(o.IPv4.DecapsulateHeader),
				api.Metadata([]byte(o.IPv4.EntryMetadata)),
				api.NHG(o.IPv4.NHG),
				api.NetworkInstance(o.IPv4.NHGNetworkInstance),
			),
		)
	case o.NH != nil:
		nheOpts := []api.GRIBIOption{
			api.Index(o.NH.Index),
			api.EncapsulateHeader(o.NH.EncapsulateHeader),
			api.DecapsulateHeader(o.NH.DecapsulateHeader),
			api.IPAddress(o.NH.IPAddress),
			api.MAC(o.NH.MAC),
			api.NetworkInstance(o.NH.NetworkInstance),
		}
		if o.NH.InterfaceReference != nil {
			if o.NH.InterfaceReference.Interface != "" {
				nheOpts = append(nheOpts, api.Interface(o.NH.InterfaceReference.Interface))
			}
			if o.NH.InterfaceReference.Subinterface != nil {
				nheOpts = append(nheOpts,
					api.SubInterface(*o.NH.InterfaceReference.Subinterface),
				)
			}
		}
		if o.NH.IPinIP != nil {
			nheOpts = append(nheOpts,
				api.IPinIP(o.NH.IPinIP.SRCIP, o.NH.IPinIP.DSTIP),
			)
		}

		for _, pmls := range o.NH.PushedMPLSLabelStack {
			nheOpts = append(nheOpts,
				api.PushedMplsLabelStack(pmls.Type, uint64(pmls.Label)),
			)
		}

		// create NH Entry Option
		opts = append(opts, api.NHEntry(nheOpts...))
	case o.NHG != nil:
		nhgeOpts := []api.GRIBIOption{
			api.ID(o.NHG.ID),
		}
		if o.NHG.BackupNHG != nil {
			nhgeOpts = append(nhgeOpts, api.BackupNextHopGroup(*o.NHG.BackupNHG))
		}
		if o.NHG.Color != nil {
			nhgeOpts = append(nhgeOpts, api.Color(*o.NHG.Color))
		}

		for _, nh := range o.NHG.NextHop {
			nhgeOpts = append(nhgeOpts, api.NHGNextHop(nh.Index, nh.Weight))
		}
		// create NHG Entry Option
		opts = append(opts, api.NHGEntry(nhgeOpts...))
	}
	return api.NewAFTOperation(opts...)
}

type ModifyInput struct {
	DefaultNetworkInstance string             `yaml:"default-network-instance" json:"default-network-instance,omitempty"`
	DefaultOperation       string             `yaml:"default-operation" json:"default-operation,omitempty"`
	Params                 *sessionParams     `yaml:"params,omitempty" json:"params,omitempty"`
	Operations             []*OperationConfig `yaml:"operations,omitempty" json:"operations,omitempty"`
}

type sessionParams struct {
	Redundancy  string `yaml:"redundancy,omitempty" json:"redundancy,omitempty"`
	Persistence string `yaml:"persistence,omitempty" json:"persistence,omitempty"`
	AckType     string `yaml:"ack-type,omitempty" json:"ack-type,omitempty"`
}

type ipv4v6Entry struct {
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// ipv4v6
	Prefix             string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	NHG                uint64 `yaml:"nhg,omitempty" json:"nhg,omitempty"`
	NHGNetworkInstance string `yaml:"nhg-network-instance,omitempty" json:"nhg-network-instance,omitempty"`
	DecapsulateHeader  string `yaml:"decapsulate-header,omitempty" json:"decapsulate-header,omitempty"`
	EntryMetadata      string `yaml:"entry-metadata,omitempty" json:"entry-metadata,omitempty"`
}

type nhgEntry struct {
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// nhg
	ID        uint64  `yaml:"id,omitempty" json:"id,omitempty"`
	BackupNHG *uint64 `yaml:"backup-nhg,omitempty" json:"backup-nhg,omitempty"`
	Color     *uint64 `yaml:"color,omitempty" json:"color,omitempty"`
	NextHop   []struct {
		Index  uint64 `yaml:"index,omitempty" json:"index,omitempty"`
		Weight uint64 `yaml:"weight,omitempty" json:"weight,omitempty"`
	} `yaml:"next-hop,omitempty" json:"next-hop,omitempty"`
	ProgrammedID *uint64 `yaml:"programmed-id,omitempty" json:"programmed-id,omitempty"`
}

type nhEntry struct {
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// nh
	Index                uint64              `yaml:"index,omitempty" json:"index,omitempty"`
	DecapsulateHeader    string              `yaml:"decapsulate-header,omitempty" json:"decapsulate-header,omitempty"`
	EncapsulateHeader    string              `yaml:"encapsulate-header,omitempty" json:"encapsulate-header,omitempty"`
	IPAddress            string              `yaml:"ip-address,omitempty" json:"ip-address,omitempty"`
	InterfaceReference   *interfaceReference `yaml:"interface-reference,omitempty" json:"interface-reference,omitempty"`
	IPinIP               *ipinip             `yaml:"ip-in-ip,omitempty" json:"ip-in-ip,omitempty"`
	MAC                  string              `yaml:"mac,omitempty" json:"mac,omitempty"`
	NetworkInstance      string              `yaml:"network-instance,omitempty" json:"network-instance,omitempty"`
	ProgrammedIndex      *uint64             `yaml:"programmed-index,omitempty" json:"programmed-index,omitempty"`
	PushedMPLSLabelStack []struct {
		Type  string `yaml:"type,omitempty" json:"type,omitempty"`
		Label uint   `yaml:"label,omitempty" json:"label,omitempty"`
	} `yaml:"pushed-mpls-label-stack,omitempty" json:"pushed-mpls-label-stack,omitempty"`
}

type interfaceReference struct {
	Interface    string  `yaml:"interface,omitempty" json:"interface,omitempty"`
	Subinterface *uint64 `yaml:"subinterface,omitempty" json:"subinterface,omitempty"`
}

type ipinip struct {
	DSTIP string `yaml:"dst-ip,omitempty" json:"dst-ip,omitempty"`
	SRCIP string `yaml:"src-ip,omitempty" json:"src-ip,omitempty"`
}

func (c *Config) GenerateModifyInputs(targetName string) (*ModifyInput, error) {
	buf := new(bytes.Buffer)
	err := c.modifyInputTemplate.Execute(buf, templateInput{
		TargetName: targetName,
		Vars:       c.modifyInputVars,
	})
	if err != nil {
		return nil, err
	}
	result := new(ModifyInput)
	err = yaml.Unmarshal(buf.Bytes(), result)
	if err != nil {
		return nil, err
	}
	sortOperations(result.Operations, "DRA")
	for i, op := range result.Operations {
		if op.NetworkInstance == "" {
			op.NetworkInstance = result.DefaultNetworkInstance
		}
		if op.Operation == "" {
			op.Operation = result.DefaultOperation
		}
		op.ID = uint64(i) + 1
		err = op.validate()
		if err != nil {
			return nil, fmt.Errorf("operation index %d is invalid: %w", op.ID, err)
		}
	}
	return result, err
}

func (c *Config) ReadModifyFileTemplate() error {
	b, err := os.ReadFile(c.ModifyInputFile)
	if err != nil {
		return err
	}
	c.modifyInputTemplate, err = utils.CreateTemplate("modify-rpc-input", string(b))
	if err != nil {
		return err
	}
	return c.readTemplateVarsFile()
}

func (c *Config) readTemplateVarsFile() error {
	if c.ModifyInputVarsFile == "" {
		ext := filepath.Ext(c.ModifyInputFile)
		c.ModifyInputVarsFile = fmt.Sprintf("%s%s%s", c.ModifyInputFile[0:len(c.ModifyInputFile)-len(ext)], varFileSuffix, ext)
		c.logger.Printf("trying to find variable file %q", c.ModifyInputVarsFile)
		_, err := os.Stat(c.ModifyInputVarsFile)
		if os.IsNotExist(err) {
			c.ModifyInputVarsFile = ""
			return nil
		} else if err != nil {
			return err
		}
	}
	b, err := readFile(c.ModifyInputVarsFile)
	if err != nil {
		return err
	}
	if c.modifyInputVars == nil {
		c.modifyInputVars = make(map[string]interface{})
	}
	err = yaml.Unmarshal(b, &c.modifyInputVars)
	if err != nil {
		return err
	}
	tempInterface := utils.Convert(c.modifyInputVars)
	switch t := tempInterface.(type) {
	case map[string]interface{}:
		c.modifyInputVars = t
	default:
		return errors.New("unexpected variables file format")
	}
	if c.Debug {
		c.logger.Printf("request vars content: %v", c.modifyInputVars)
	}
	return nil
}

func ParseUint128(v string) (*spb.Uint128, error) {
	if v == "" {
		return nil, nil
	}
	if strings.HasPrefix(v, ":") {
		v = "0" + v
	}
	if strings.HasSuffix(v, ":") {
		v = v + "0"
	}

	lh := strings.SplitN(v, ":", 2)
	switch len(lh) {
	case 1:
		vi, err := strconv.Atoi(lh[0])
		if err != nil {
			return nil, err
		}
		return &spb.Uint128{Low: uint64(vi)}, nil
	case 2:
		if lh[0] == "" {
			lh[0] = "0"
		}
		v0i, err := strconv.Atoi(lh[0])
		if err != nil {
			return nil, err
		}
		if lh[1] == "" {
			lh[1] = "0"
		}
		v1i, err := strconv.Atoi(lh[1])
		if err != nil {
			return nil, err
		}
		return &spb.Uint128{High: uint64(v0i), Low: uint64(v1i)}, nil
	}
	return nil, nil
}

// readFile reads a json or yaml file. the the file is .yaml, converts it to json and returns []byte and an error
func readFile(name string) ([]byte, error) {
	data, err := utils.ReadFile(context.TODO(), name)
	if err != nil {
		return nil, err
	}
	//
	switch filepath.Ext(name) {
	case ".json":
		return data, err
	case ".yaml", ".yml":
		return tryYAML(data)
	default:
		// try yaml
		newData, err := tryYAML(data)
		if err != nil {
			// assume json
			return data, nil
		}
		return newData, nil
	}
}

func tryYAML(data []byte) ([]byte, error) {
	var out interface{}
	var err error
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	newStruct := utils.Convert(out)
	newData, err := json.Marshal(newStruct)
	if err != nil {
		return nil, err
	}
	return newData, nil
}

type templateInput struct {
	TargetName string
	Vars       map[string]interface{}
}

// sortOperations sorts the given []*OperationConfig slice based on the ordering type
// string. For example DRA = DELETE, REPLACE, ADD
func sortOperations(ops []*OperationConfig, order string) {
	switch strings.ToUpper(order) {
	case "DRA":
		sortOperationsDRA(ops)
	case "DAR":
		sortOperationsDAR(ops)
	case "ARD":
		sortOperationsARD(ops)
	case "ADR":
		sortOperationsADR(ops)
	case "RAD":
		sortOperationsRAD(ops)
	case "RDA":
		sortOperationsRDA(ops)
	}
}

// sortOperationsDRA sorts the given []*OperationConfig by operation type then by entry type.
// Operation type sort order is: Deletes, Replaces then Additions.
// within Deletes: IPv4/v6 entries are sent first then NHGs and finally NHs.
// within Replaces or Additions: NH are sent first, then NHGs and last are IPv4/v6 entries
func sortOperationsDRA(ops []*OperationConfig) {
	sort.SliceStable(ops, func(i, j int) bool {
		switch strings.ToUpper(ops[i].Operation) {
		case "DELETE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return lessDeleteOp(ops[i], ops[j])
			case "ADD":
				return true
			case "REPLACE":
				return true
			}
		case "REPLACE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return false
			case "ADD":
				return true
			case "REPLACE":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		case "ADD":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return false
			case "REPLACE":
				return false
			case "ADD":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		}
		return false
	})
}

// sortOperationsDAR sorts the given []*OperationConfig by operation type then by entry type.
// Operation type sort order is: Deletes, Additions then Replaces.
// within Deletes: IPv4/v6 entries are sent first then NHGs and finally NHs.
// within Replaces or Additions: NH are sent first, then NHGs and last are IPv4/v6 entries
func sortOperationsDAR(ops []*OperationConfig) {
	sort.Slice(ops, func(i, j int) bool {
		switch strings.ToUpper(ops[i].Operation) {
		case "DELETE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return lessDeleteOp(ops[i], ops[j])
			case "ADD":
				return true
			case "REPLACE":
				return true
			}
		case "REPLACE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return false
			case "ADD":
				return true
			case "REPLACE":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		case "ADD":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return false
			case "REPLACE":
				return true
			case "ADD":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		}
		return false
	})
}

// sortOperationsARD sorts the given []*OperationConfig by operation type then by entry type.
// Operation type sort order is: Additions, Replaces then Deletes.
// within Deletes: IPv4/v6 entries are sent first then NHGs and finally NHs.
// within Replaces or Additions: NH are sent first, then NHGs and last are IPv4/v6 entries
func sortOperationsARD(ops []*OperationConfig) {
	sort.Slice(ops, func(i, j int) bool {
		switch strings.ToUpper(ops[i].Operation) {
		case "DELETE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return lessDeleteOp(ops[i], ops[j])
			case "ADD":
				return true
			case "REPLACE":
				return true
			}
		case "REPLACE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return true
			case "ADD":
				return false
			case "REPLACE":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		case "ADD":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return true
			case "REPLACE":
				return true
			case "ADD":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		}
		return false
	})
}

// sortOperationsADR sorts the given []*OperationConfig by operation type then by entry type.
// Operation type sort order is: Additions, Deletes then Replaces.
// within Deletes: IPv4/v6 entries are sent first then NHGs and finally NHs.
// within Replaces or Additions: NH are sent first, then NHGs and last are IPv4/v6 entries
func sortOperationsADR(ops []*OperationConfig) {
	sort.Slice(ops, func(i, j int) bool {
		switch strings.ToUpper(ops[i].Operation) {
		case "DELETE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return lessDeleteOp(ops[i], ops[j])
			case "ADD":
				return false
			case "REPLACE":
				return true
			}
		case "REPLACE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return false
			case "ADD":
				return false
			case "REPLACE":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		case "ADD":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return true
			case "REPLACE":
				return true
			case "ADD":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		}
		return false
	})
}

// sortOperationsRAD sorts the given []*OperationConfig by operation type then by entry type.
// Operation type sort order is: Replaces, Additions then Deletes.
// within Deletes: IPv4/v6 entries are sent first then NHGs and finally NHs.
// within Replaces or Additions: NH are sent first, then NHGs and last are IPv4/v6 entries
func sortOperationsRAD(ops []*OperationConfig) {
	sort.Slice(ops, func(i, j int) bool {
		switch strings.ToUpper(ops[i].Operation) {
		case "DELETE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return lessDeleteOp(ops[i], ops[j])
			case "ADD":
				return false
			case "REPLACE":
				return false
			}
		case "REPLACE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return true
			case "ADD":
				return true
			case "REPLACE":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		case "ADD":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return true
			case "REPLACE":
				return false
			case "ADD":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		}
		return false
	})
}

// sortOperationsRDA sorts the given []*OperationConfig by operation type then by entry type.
// Operation type sort order is: Replaces, Deletes then Additions.
// within Deletes: IPv4/v6 entries are sent first then NHGs and finally NHs.
// within Replaces or Additions: NH are sent first, then NHGs and last are IPv4/v6 entries
func sortOperationsRDA(ops []*OperationConfig) {
	sort.Slice(ops, func(i, j int) bool {
		switch strings.ToUpper(ops[i].Operation) {
		case "DELETE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return lessDeleteOp(ops[i], ops[j])
			case "ADD":
				return true
			case "REPLACE":
				return false
			}
		case "REPLACE":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return true
			case "ADD":
				return true
			case "REPLACE":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		case "ADD":
			switch strings.ToUpper(ops[j].Operation) {
			case "DELETE":
				return false
			case "REPLACE":
				return false
			case "ADD":
				return lessAddOrReplaceOp(ops[i], ops[j])
			}
		}
		return false
	})
}

func lessAddOrReplaceOp(op1, op2 *OperationConfig) bool {
	switch {
	case op1.NH != nil:
		return true
	case op1.NHG != nil:
		switch {
		case op2.NH != nil:
			return false
		default:
			return true
		}
	case op1.IPv4 != nil:
		switch {
		case op2.NH != nil, op2.NHG != nil:
			return false
		default:
			return true
		}
	case op1.IPv6 != nil:
		switch {
		case op2.NH != nil, op2.NHG != nil:
			return false
		case op2.IPv4 != nil:
			return false
		default:
			return true
		}
	default:
		return true
	}
}

func lessDeleteOp(op1, op2 *OperationConfig) bool {
	switch {
	case op1.IPv6 != nil, op1.IPv4 != nil:
		switch {
		case op2.NH != nil, op2.NHG != nil:
			return true
		default: // ipv4/v6
			return false
		}
	case op1.NHG != nil:
		switch {
		case op2.IPv4 != nil, op2.IPv6 != nil:
			return false
		case op2.NH != nil:
			return true
		default: // nhg
			return true
		}
	case op1.NH != nil:
		switch {
		case op2.NH != nil:
			return true
		default:
			return false
		}
	default:
		return true
	}
}
