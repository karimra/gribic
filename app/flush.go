package app

import (
	"context"
	"fmt"

	"github.com/karimra/gribic/api"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/prototext"
)

type flushResponse struct {
	TargetError
	// req *spb.FlushResponse
	rsp *spb.FlushResponse
}

func (a *App) InitFlushFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	//
	cmd.Flags().StringVarP(&a.Config.FlushNetworkInstance, "ns", "", "", "network instance name")
	cmd.Flags().BoolVarP(&a.Config.FlushNetworkInstanceAll, "ns-all", "", false, "run Get against all network instance(s)")

	cmd.Flags().BoolVarP(&a.Config.FlushElectionIDOverride, "override", "", false, "override election ID")
	//
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) FlushPreRunE(cmd *cobra.Command, args []string) error {
	// parse election ID
	var err error
	a.electionID, err = parseUint128(a.Config.GlobalFlags.ElectionID)
	if err != nil {
		return err
	}
	// TODO: validate flags
	return nil
}

func (a *App) FlushRunE(cmd *cobra.Command, args []string) error {
	targets, err := a.GetTargets()
	if err != nil {
		return err
	}
	a.Logger.Debugf("targets: %v", targets)
	numTargets := len(targets)
	responseChan := make(chan *flushResponse, numTargets)

	a.wg.Add(numTargets)
	for _, t := range targets {
		go func(t *target) {
			defer a.wg.Done()
			// create context
			ctx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			// append credentials to context
			ctx = metadata.AppendToOutgoingContext(ctx, "username", *t.Config.Username, "password", *t.Config.Password)
			// create a grpc conn
			err = a.CreateGrpcClient(ctx, t, a.createBaseDialOpts()...)
			if err != nil {
				responseChan <- &flushResponse{
					TargetError: TargetError{
						TargetName: t.Config.Address,
						Err:        err,
					},
				}
				return
			}
			defer t.Close()
			rsp, err := a.gribiFlush(ctx, t)
			responseChan <- &flushResponse{
				TargetError: TargetError{
					TargetName: t.Config.Address,
					Err:        err,
				},
				rsp: rsp,
			}
		}(t)
	}
	//
	a.wg.Wait()
	close(responseChan)

	errs := make([]error, 0) //, numTargets)
	result := make([]*flushResponse, 0, numTargets)
	for rsp := range responseChan {
		if rsp.Err != nil {
			wErr := fmt.Errorf("%q Flush RPC failed: %v", rsp.TargetName, rsp.Err)
			a.Logger.Error(wErr)
			errs = append(errs, wErr)
			continue
		}
		result = append(result, rsp)
	}
	a.Logger.Printf("got %d results", len(result))
	for _, r := range result {
		a.Logger.Infof("%q: %s", r.TargetName, prototext.Format(r.rsp))
	}
	return a.handleErrs(errs)
}

func (a *App) gribiFlush(ctx context.Context, t *target) (*spb.FlushResponse, error) {
	opts := make([]api.GRIBIOption, 0, 2)
	switch {
	case a.Config.FlushNetworkInstanceAll:
		opts = append(opts, api.NSAll())
	default:
		opts = append(opts, api.NetworkInstance(a.Config.FlushNetworkInstance))
	}
	switch {
	case a.Config.FlushElectionIDOverride:
		opts = append(opts, api.Override())
	default:
		opts = append(opts, api.ElectionID(a.electionID))
	}
	req, err := api.NewFlushRequest(opts...)
	if err != nil {
		return nil, err
	}
	a.Logger.Debugf("target %s request:\n%s", t.Config.Name, prototext.Format(req))
	t.gRIBIClient = spb.NewGRIBIClient(t.conn)
	return a.flush(ctx, t, req)
}

func (a *App) flush(ctx context.Context, t *target, req *spb.FlushRequest) (*spb.FlushResponse, error) {
	return t.gRIBIClient.Flush(ctx, req)
}
