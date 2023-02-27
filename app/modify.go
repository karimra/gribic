package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/karimra/gribic/api"
	"github.com/karimra/gribic/config"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"
)

type modifyResponse struct {
	TargetError
	rsp *spb.ModifyResponse
}

func (a *App) InitModifyFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	// session redundancy
	// cmd.Flags().BoolVarP(&a.Config.ModifySessionRedundancyAllPrimary, "all-primary", "", false, "set session client redundancy to ALL_PRIMARY")
	cmd.Flags().BoolVarP(&a.Config.ModifySessionRedundancySinglePrimary, "single-primary", "", false, "set session client redundancy to SINGLE_PRIMARY")
	// session persistence
	// cmd.Flags().BoolVarP(&a.Config.ModifySessionPersistanceDelete, "delete", "", false, "set session persistence to DELETE")
	cmd.Flags().BoolVarP(&a.Config.ModifySessionPersistancePreserve, "preserve", "", false, "set session persistence to PRESERVE")
	// session ack
	// cmd.Flags().BoolVarP(&a.Config.ModifySessionRibAck, "rib", "", false, "set session ack type to RIB")
	cmd.Flags().BoolVarP(&a.Config.ModifySessionRibFibAck, "fib", "", false, "set session ack type to RIB_FIB")
	// modify input file
	cmd.Flags().StringVarP(&a.Config.ModifyInputFile, "input-file", "", "", "path to a file specifying the modify RPC input")
}

func (a *App) ModifyPreRunE(cmd *cobra.Command, args []string) error {
	// parse election ID
	var err error
	a.electionID, err = config.ParseUint128(a.Config.ElectionID)
	if err != nil {
		return err
	}
	if a.Config.ModifyInputFile == "" {
		return errors.New("missing --input-file value")
	}

	err = a.Config.ReadModifyFileTemplate()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) ModifyRunE(cmd *cobra.Command, args []string) error {
	targets, err := a.GetTargets()
	if err != nil {
		return err
	}
	a.Logger.Debugf("targets: %v", targets)
	numTargets := len(targets)
	responseChan := make(chan *modifyResponse, numTargets)
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
				responseChan <- &modifyResponse{
					TargetError: TargetError{
						TargetName: t.Config.Address,
						Err:        err,
					},
				}
				return
			}
			defer t.Close()
			rspCh := a.gribiModify(ctx, t)
			for {
				select {
				case rsp, ok := <-rspCh:
					if !ok {
						return
					}
					if rsp != nil {
						if rsp.TargetError.Err != nil {
							a.Logger.Errorf("%+v", rsp.TargetError)
						} else {
							a.Logger.Printf("%s\nresponse: %s", rsp.TargetError.TargetName, prototext.Format(rsp.rsp))
						}

					}
				case <-ctx.Done():
					a.Logger.Print(ctx.Err())
				}
			}
		}(t)
	}
	//
	a.wg.Wait()
	close(responseChan)

	return nil
}

func (a *App) gribiModify(ctx context.Context, t *target) chan *modifyResponse {
	rspCh := make(chan *modifyResponse)
	t.gRIBIClient = spb.NewGRIBIClient(t.conn)

	go func() {
		defer func() {
			close(rspCh)
			a.Logger.Infof("target %s modify stream done", t.Config.Name)
		}()
		// create client
		modClient, err := t.gRIBIClient.Modify(ctx)
		if err != nil {
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
			}
			return
		}
		modifyInput, err := a.Config.GenerateModifyInputs(t.Config.Name)
		if err != nil {
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
			}
			return
		}

		// session parameters & election ID
		modParams, err := a.createModifyRequestParams(modifyInput)
		if err != nil {
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
			}
			return
		}
		switch len(modParams) {
		case 1: // no election ID (all-primary)
			a.Logger.Printf("sending request=%v to %q", modParams[0], t.Config.Name)
			err = modClient.Send(modParams[0])
			if err != nil {
				rspCh <- &modifyResponse{
					TargetError: TargetError{
						TargetName: t.Config.Name,
						Err:        err,
					},
				}
				return
			}
			modRsp, err := modClient.Recv()
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
				rsp: modRsp,
			}
			if err != nil {
				return
			}
		case 2: // session params and electionID (single-primary)
			a.Logger.Printf("sending request=%v to %q", modParams[0], t.Config.Name)
			err = modClient.Send(modParams[0])
			if err != nil {
				rspCh <- &modifyResponse{
					TargetError: TargetError{
						TargetName: t.Config.Name,
						Err:        err,
					},
				}
				return
			}
			modRsp, err := modClient.Recv()
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
				rsp: modRsp,
			}
			if err != nil {
				return
			}

			// send election ID
			a.Logger.Printf("sending request=%v to %q", modParams[1], t.Config.Name)
			err = modClient.Send(modParams[1])
			if err != nil {
				rspCh <- &modifyResponse{
					TargetError: TargetError{
						TargetName: t.Config.Name,
						Err:        err,
					},
				}
				return
			}
			modRsp, err = modClient.Recv()
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
				rsp: modRsp,
			}
			if err != nil {
				return
			}
			if a.electionID != nil && modRsp.ElectionId != nil {
				if a.electionID.High < modRsp.ElectionId.High {
					a.Logger.Infof("target's last known electionID is higher than client's: %+v > %+v", modRsp.ElectionId, a.electionID)
					return
				}
				if a.electionID.High == modRsp.ElectionId.High && a.electionID.Low < modRsp.ElectionId.Low {
					a.Logger.Infof("target's last known electionID is higher than client's: %+v > %+v", modRsp.ElectionId, a.electionID)
					return
				}
			}
		}
		modReqs, err := a.createModifyRequestOperation(modifyInput)
		if err != nil {
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
			}
			return
		}
		// operations
		for _, req := range modReqs {
			a.Logger.Infof("target %s modify request:\n%s", t.Config.Name, prototext.Format(req))
			err = modClient.Send(req)
			if err != nil {
				rspCh <- &modifyResponse{
					TargetError: TargetError{
						TargetName: t.Config.Name,
						Err:        err,
					},
				}
				return
			}
			modRsp, err := modClient.Recv()
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
				rsp: modRsp,
			}
			if err != nil {
				return
			}
			for _, result := range modRsp.GetResult() {
				switch result.GetStatus() {
				case spb.AFTResult_UNSET: // TODO: consider this an error ?
				// case spb.AFTResult_OK: DEPRECATED
				case spb.AFTResult_FAILED:
					return
				case spb.AFTResult_RIB_PROGRAMMED:
				case spb.AFTResult_FIB_PROGRAMMED:
				case spb.AFTResult_FIB_FAILED:
					return
				}
			}
		}
	}()

	return rspCh
}

func (a *App) createModifyRequestParams(modifyInput *config.ModifyInput) ([]*spb.ModifyRequest, error) {
	if modifyInput.Params == nil {
		modReq, err := api.NewModifyRequest(
			api.PersistenceDelete(),
			api.RedundancyAllPrimary(),
			api.AckTypeRib(),
		)
		return []*spb.ModifyRequest{modReq}, err
	}

	opts := make([]api.GRIBIOption, 0, 4)

	switch {
	case a.Config.ModifySessionPersistancePreserve ||
		(modifyInput.Params.Persistence == "preserve" && !a.Config.ModifySessionPersistancePreserve):
		opts = append(opts, api.PersistencePreserve())
	default:
		opts = append(opts, api.PersistenceDelete())
	}

	switch {
	case a.Config.ModifySessionRedundancySinglePrimary ||
		(modifyInput.Params.Redundancy == "single-primary" && !a.Config.ModifySessionRedundancySinglePrimary):
		opts = append(opts,
			api.RedundancySinglePrimary(),
			// ,
		)
	default:
		opts = append(opts, api.RedundancyAllPrimary())
	}

	switch {
	case a.Config.ModifySessionRibFibAck ||
		(modifyInput.Params.AckType == "rib-fib" && !a.Config.ModifySessionRibFibAck):
		opts = append(opts, api.AckTypeRibFib())
	default:
		opts = append(opts, api.AckTypeRib())
	}
	sessParams, err := api.NewModifyRequest(opts...)
	if err != nil {
		return nil, err
	}
	elecIdReq, err := api.NewModifyRequest(api.ElectionID(a.electionID))
	if err != nil {
		return nil, err
	}
	return []*spb.ModifyRequest{sessParams, elecIdReq}, err
}

func (a *App) createModifyRequestOperation(modifyInput *config.ModifyInput) ([]*spb.ModifyRequest, error) {
	reqs := make([]*spb.ModifyRequest, 0)

	for _, op := range modifyInput.Operations {
		req := new(spb.ModifyRequest)
		aftOp, err := op.CreateAftOper()
		if err != nil {
			return nil, err
		}
		req.Operation = append(req.Operation, aftOp)
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (a *App) modifyChan(ctx context.Context, t *target, modReqCh chan *spb.ModifyRequest) (chan *spb.ModifyResponse, chan error) {
	rspChan := make(chan *spb.ModifyResponse)
	errChan := make(chan error)

	go func() {
		defer close(rspChan)
		defer close(errChan)
		// stream sending goroutine
		go func() {
			var err error
			for {
				select {
				case <-ctx.Done():
					return
				case req, ok := <-modReqCh:
					if !ok {
						return
					}
					err = t.modClient.Send(req)
					if err != nil {
						errChan <- fmt.Errorf("failed sending request: %v: err=%v", req, err)
						return
					}
				}
			}
		}()
		// receive stream
		for {
			modRsp, err := t.modClient.Recv()
			if err != nil {
				errChan <- err
				return
			}
			rspChan <- modRsp
		}
	}()

	return rspChan, errChan
}
