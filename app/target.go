package app

import (
	"context"

	"github.com/karimra/gribic/config"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/openconfig/gribigo/rib"
	"google.golang.org/grpc"
)

type target struct {
	// configuration
	Config *config.TargetConfig
	// gRPC connection
	conn *grpc.ClientConn
	// gRIBI client
	gRIBIClient spb.GRIBIClient
	// modify stream client
	modClient spb.GRIBI_ModifyClient
	// cancel function
	cfn context.CancelFunc
	// modify stream cancel function
	modifyCfn context.CancelFunc
	// RIB
	rib *rib.RIB
}

func NewTarget(tc *config.TargetConfig) *target {
	return &target{
		Config: tc,
		rib:    rib.New(tc.DefaultNI),
	}
}

func (a *App) GetTargets() (map[string]*target, error) {
	targetsConfigs, err := a.Config.GetTargets()
	if err != nil {
		return nil, err
	}
	targets := make(map[string]*target)
	for n, tc := range targetsConfigs {
		targets[n] = NewTarget(tc)
	}
	return targets, nil
}

func (t *target) Close() error {
	if t.conn == nil {
		return nil
	}
	return t.conn.Close()
}

func (t *target) createModifyClient(ctx context.Context) error {
	mctx, cancel := context.WithCancel(ctx)
	t.modifyCfn = cancel
	var err error
	t.modClient, err = t.gRIBIClient.Modify(appendCredentials(mctx, t.Config))
	return err
}
