package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karimra/gnmic/utils"
	"github.com/karimra/gribic/api"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"
)

type Workflow struct {
	Name  string  `yaml:"name,omitempty"`
	Steps []*step `yaml:"steps,omitempty"`
}

type step struct {
	Name      string        `yaml:"name,omitempty"`
	Wait      time.Duration `yaml:"wait,omitempty"`
	WaitAfter time.Duration `yaml:"wait-after,omitempty"`
	// determines the RPC type
	RPC string `yaml:"rpc,omitempty"`
	// network instance, applies if RPC is "get" or "flush"
	NetworkInstance string `yaml:"network-instance,omitempty"`
	// AFT type, applies if RPC is "get" or "flush"
	Aft string `yaml:"aft,omitempty"`
	// Override, applies if RPC is "flush"
	Override bool `yaml:"override,omitempty"`
	// Session Parameters for "modify RPC"
	SessionParams *sessionParams `yaml:"session-params,omitempty"`
	// ElectionID for "modify" with session parameters and "flush" RPCs
	ElectionID string `yaml:"election-id,omitempty"`
	// Operations for "modify" RPC
	Operations []*OperationConfig `yaml:"operations,omitempty"`
}

func (s *step) BuildRequests() ([]proto.Message, error) {
	switch strings.ToLower(s.RPC) {
	case "get":
		return s.buildGetRequest()
	case "flush":
		return s.buildFlushRequest()
	case "modify":
		return s.buildModifyRequest()
	}
	return nil, nil
}

func (s *step) buildGetRequest() ([]proto.Message, error) {
	opts := make([]api.GRIBIOption, 0)
	if s.NetworkInstance == "" {
		opts = append(opts, api.NSAll())
	} else {
		opts = append(opts, api.NetworkInstance(s.NetworkInstance))
	}
	if s.Aft == "" {
		opts = append(opts, api.AFTTypeAll())
	} else {
		opts = append(opts, api.AFTType(s.Aft))
	}

	req, err := api.NewGetRequest(opts...)
	if err != nil {
		return nil, err
	}
	return []proto.Message{req}, nil
}

func (s *step) buildFlushRequest() ([]proto.Message, error) {
	opts := make([]api.GRIBIOption, 0, 2)
	if s.NetworkInstance == "" {
		opts = append(opts, api.NSAll())
	} else {
		opts = append(opts, api.NetworkInstance(s.NetworkInstance))
	}
	if s.Override {
		opts = append(opts, api.Override())
	} else {
		eID, err := ParseUint128(s.ElectionID)
		if err != nil {
			return nil, err
		}
		opts = append(opts, api.ElectionID(eID))
	}
	req, err := api.NewFlushRequest(opts...)
	if err != nil {
		return nil, err
	}
	return []proto.Message{req}, nil
}

func (s *step) buildModifyRequest() ([]proto.Message, error) {
	reqs := make([]proto.Message, 0, 2)
	opts := make([]api.GRIBIOption, 0, 4)
	// fmt.Println(s)
	if s.SessionParams != nil {
		// persistence
		if strings.ToLower(s.SessionParams.Persistence) == "preserve" {
			opts = append(opts, api.PersistencePreserve())
		}
		// redundancy
		if strings.ToLower(s.SessionParams.Redundancy) == "single-primary" {
			eID, err := ParseUint128(s.ElectionID)
			if err != nil {
				return nil, err
			}
			opts = append(opts,
				api.RedundancySinglePrimary(),
				api.ElectionID(eID),
			)
		}
		// ack
		if s.SessionParams.AckType == "rib-fib" {
			opts = append(opts, api.AckTypeRibFib())
		}

		req, err := api.NewModifyRequest(opts...)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	// aft modify if any
	if len(s.Operations) > 0 {
		req := &spb.ModifyRequest{
			Operation: make([]*spb.AFTOperation, 0, len(s.Operations)),
		}
		for _, op := range s.Operations {
			spbOp, err := op.CreateAftOper()
			if err != nil {
				return nil, err
			}
			req.Operation = append(req.Operation, spbOp)
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (c *Config) ReadWorkflowFile() error {
	c.logger.Infof("reading workflow file: %s", c.WorkflowFile)
	b, err := os.ReadFile(c.WorkflowFile)
	if err != nil {
		return err
	}

	c.workflowTemplate, err = utils.CreateTemplate("workflow-template", string(b))
	if err != nil {
		return err
	}
	return c.readWorkflowTemplateVarsFile()
}

func (c *Config) readWorkflowTemplateVarsFile() error {
	if c.WorkflowInputVarsFile == "" {
		ext := filepath.Ext(c.WorkflowFile)
		c.WorkflowInputVarsFile = fmt.Sprintf("%s%s%s", c.WorkflowFile[0:len(c.WorkflowFile)-len(ext)], varFileSuffix, ext)
		c.logger.Debugf("trying to find variable file %q", c.WorkflowInputVarsFile)
		_, err := os.Stat(c.WorkflowInputVarsFile)
		if os.IsNotExist(err) {
			c.WorkflowInputVarsFile = ""
			return nil
		} else if err != nil {
			return err
		}
	}
	b, err := readFile(c.WorkflowInputVarsFile)
	if err != nil {
		return err
	}
	if c.workflowVars == nil {
		c.workflowVars = make(map[string]interface{})
	}
	err = yaml.Unmarshal(b, &c.workflowVars)
	if err != nil {
		return err
	}
	tempInterface := utils.Convert(c.workflowVars)
	switch t := tempInterface.(type) {
	case map[string]interface{}:
		c.workflowVars = t
	default:
		return errors.New("unexpected variables file format")
	}
	if c.Debug {
		c.logger.Printf("request vars content: %v", c.workflowVars)
	}
	fmt.Printf("workflow vars: %v\n", c.workflowVars)
	return nil
}

func (c *Config) GenerateWorkflow(targetName string) (*Workflow, error) {
	buf := new(bytes.Buffer)
	err := c.workflowTemplate.Execute(buf,
		templateInput{
			TargetName: targetName,
			Vars:       c.workflowVars,
		},
	)
	if err != nil {
		return nil, err
	}
	wf := new(Workflow)
	err = yaml.Unmarshal(buf.Bytes(), wf)
	// fmt.Printf("workflow for target=%q: %+v\n", targetName, wf)
	return wf, err
}
