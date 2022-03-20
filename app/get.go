package app

import (
	"context"
	"fmt"
	"io"

	"github.com/karimra/gribic/api"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/prototext"
)

type getResponse struct {
	TargetError
	rsp []*spb.GetResponse
}

func (a *App) InitGetFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	//
	cmd.Flags().StringVarP(&a.Config.GetNetworkInstance, "ns", "", "", "network instance name, an empty network-instance name means query all instances.")
	cmd.Flags().StringVarP(&a.Config.GetAFT, "aft", "", "ALL", "AFT type, one of: ALL, IPv4, IPv6, NH, NHG, MPLS, MAC or PF")

	//
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) GetRunE(cmd *cobra.Command, args []string) error {
	targets, err := a.GetTargets()
	if err != nil {
		return err
	}
	a.Logger.Debugf("targets: %v", targets)
	numTargets := len(targets)
	responseChan := make(chan *getResponse, numTargets)

	a.wg.Add(numTargets)
	for _, t := range targets {
		go func(t *target) {
			defer a.wg.Done()
			// create context
			ctx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			// append credentials to context
			ctx = appendCredentials(ctx, t.Config)
			// create a grpc conn
			err = a.CreateGrpcClient(ctx, t, a.createBaseDialOpts()...)
			if err != nil {
				responseChan <- &getResponse{
					TargetError: TargetError{
						TargetName: t.Config.Address,
						Err:        err,
					},
				}
				return
			}
			defer t.Close()
			rsp, err := a.gribiGet(ctx, t)
			responseChan <- &getResponse{
				TargetError: TargetError{
					TargetName: t.Config.Address,
					Err:        err,
				},
				rsp: []*spb.GetResponse{rsp},
			}
		}(t)
	}
	//
	a.wg.Wait()
	close(responseChan)

	errs := make([]error, 0) //, numTargets)
	result := make([]*getResponse, 0, numTargets)
	for rsp := range responseChan {
		if rsp.Err != nil {
			wErr := fmt.Errorf("%q Get RPC failed: %v", rsp.TargetName, rsp.Err)
			a.Logger.Error(wErr)
			errs = append(errs, wErr)
			continue
		}
		result = append(result, rsp)
	}
	a.Logger.Printf("got %d results", len(result))
	for _, r := range result {
		for _, gr := range r.rsp {
			a.Logger.Infof("%q:\n%v", r.TargetName, prototext.Format(gr))
		}
	}
	return a.handleErrs(errs)
}

func (a *App) gribiGet(ctx context.Context, t *target) (*spb.GetResponse, error) {
	opts := make([]api.GRIBIOption, 0, 2)
	opts = append(opts, api.AFTType(a.Config.GetAFT))
	if a.Config.GetNetworkInstance == "" {
		opts = append(opts, api.NSAll())
	} else {
		opts = append(opts, api.NetworkInstance(a.Config.GetNetworkInstance))
	}

	req, err := api.NewGetRequest(opts...)
	if err != nil {
		return nil, err
	}
	t.gRIBIClient = spb.NewGRIBIClient(t.conn)
	return a.get(ctx, t, req)
}

func (a *App) get(ctx context.Context, t *target, req *spb.GetRequest) (*spb.GetResponse, error) {
	stream, err := t.gRIBIClient.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	resp := &spb.GetResponse{
		Entry: make([]*spb.AFTEntry, 0),
	}
	for {
		getres, err := stream.Recv()
		if err == io.EOF {
			a.Logger.Debugf("target %s: received EOF", t.Config.Name)
			break
		}
		if err != nil {
			return nil, err
		}
		a.Logger.Debugf("target %s: intermediate get response: %v", t.Config.Name, getres)
		resp.Entry = append(resp.Entry, getres.GetEntry()...)
	}
	a.Logger.Infof("target %s: final get response: %+v", t.Config.Name, resp)
	return resp, nil
}

func (a *App) getChan(ctx context.Context, t *target, req *spb.GetRequest) (chan *spb.GetResponse, chan error) {
	rspChan := make(chan *spb.GetResponse)
	errChan := make(chan error)
	go func() {
		defer close(rspChan)
		defer close(errChan)
		stream, err := t.gRIBIClient.Get(ctx, req)
		if err != nil {
			errChan <- err
			return
		}
		for {
			getres, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}
			rspChan <- getres
		}
	}()
	return rspChan, errChan
}
