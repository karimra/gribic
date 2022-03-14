package app

import (
	"context"

	"github.com/karimra/gribic/api"
	"github.com/karimra/gribic/config"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
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
	a.electionID, err = parseUint128(a.Config.ElectionID)
	if err != nil {
		return err
	}
	// TODO: validate flags
	//
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
			ctx = metadata.AppendToOutgoingContext(ctx, "username", *t.Config.Username, "password", *t.Config.Password)
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
						a.Logger.Printf("%+v\n response: %s", rsp.TargetError, prototext.Format(rsp.rsp))
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
		// session parameters
		modParams, err := a.createModifyRequestParams()
		if err != nil {
			rspCh <- &modifyResponse{
				TargetError: TargetError{
					TargetName: t.Config.Name,
					Err:        err,
				},
			}
			return
		}
		a.Logger.Printf("sending request=%v to %q", modParams, t.Config.Name)
		err = modClient.Send(modParams)
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
		// TODO: compare returned session params and electionID
		modReqs, err := a.createModifyRequestOperation()
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

func (a *App) createModifyRequestParams() (*spb.ModifyRequest, error) {
	opts := make([]api.GRIBIOption, 0, 4)
	fileInput, err := config.ReadModifyFile(a.Config.ModifyInputFile)
	if err != nil {
		return nil, err
	}

	if a.Config.ModifySessionPersistancePreserve ||
		(fileInput.Params.Persistence == "preserve" && !a.Config.ModifySessionPersistancePreserve) {
		opts = append(opts, api.PersistencePreserve())
	}
	if a.Config.ModifySessionRedundancySinglePrimary ||
		(fileInput.Params.Redundancy == "single-primary" && !a.Config.ModifySessionRedundancySinglePrimary) {
		opts = append(opts,
			api.RedundancySinglePrimary(),
			api.ElectionID(a.electionID),
		)
	}
	if a.Config.ModifySessionRibFibAck ||
		(fileInput.Params.AckType == "rib-fib" && !a.Config.ModifySessionRibFibAck) {
		opts = append(opts, api.AckTypeRibFib())
	}
	return api.NewModifyRequest(opts...)
}

func (a *App) createModifyRequestOperation() ([]*spb.ModifyRequest, error) {
	reqs := make([]*spb.ModifyRequest, 0)
	ops, err := config.ReadModifyFile(a.Config.ModifyInputFile)
	if err != nil {
		return nil, err
	}
	for _, op := range ops.Operations {
		req := new(spb.ModifyRequest)
		aftOp, err := a.createAftOper(op)
		if err != nil {
			return nil, err
		}
		req.Operation = append(req.Operation, aftOp)
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (a *App) createAftOper(opc *config.OperationConfig) (*spb.AFTOperation, error) {
	opts := []api.GRIBIOption{
		api.ID(opc.ID),
		api.NetworkInstance(opc.NetworkInstance),
		api.Op(opc.Operation),
		api.ElectionID(a.electionID),
	}

	switch {
	case opc.IPv6 != nil:
		// append IPv6Entry option
		opts = append(opts,
			api.IPv6Entry(
				api.Prefix(opc.IPv6.Prefix),
				api.DecapsulateHeader(opc.IPv6.DecapsulateHeader),
				api.Metadata([]byte(opc.IPv6.EntryMetadata)),
				api.NHG(opc.IPv6.NHG),
				api.NetworkInstance(opc.IPv6.NHGNetworkInstance),
			),
		)
	case opc.IPv4 != nil:
		// append IPv4Entry option
		opts = append(opts,
			api.IPv4Entry(
				api.Prefix(opc.IPv4.Prefix),
				api.DecapsulateHeader(opc.IPv4.DecapsulateHeader),
				api.Metadata([]byte(opc.IPv4.EntryMetadata)),
				api.NHG(opc.IPv4.NHG),
				api.NetworkInstance(opc.IPv4.NHGNetworkInstance),
			),
		)
	case opc.NH != nil:
		nheOpts := []api.GRIBIOption{
			api.Index(opc.NH.Index),
			api.EncapsulateHeader(opc.NH.EncapsulateHeader),
			api.DecapsulateHeader(opc.NH.DecapsulateHeader),
			api.IPAddress(opc.NH.IPAddress),
			//
			api.MAC(opc.NH.MAC),
			api.NetworkInstance(opc.NH.NetworkInstance),
		}
		if opc.NH.InterfaceReference != nil {
			if opc.NH.InterfaceReference.Interface != "" {
				nheOpts = append(nheOpts, api.Interface(opc.NH.InterfaceReference.Interface))
			}
			if opc.NH.InterfaceReference.Subinterface != nil {
				nheOpts = append(nheOpts,
					api.SubInterface(*opc.NH.InterfaceReference.Subinterface),
				)
			}
		}
		if opc.NH.IPinIP != nil {
			nheOpts = append(nheOpts,
				api.IPinIP(opc.NH.IPinIP.SRCIP, opc.NH.IPinIP.DSTIP),
			)
		}
		if opc.NH.ProgrammedIndex != nil {
			nheOpts = append(nheOpts, api.ProgrammedIndex(*opc.NH.ProgrammedIndex))
		}

		for _, pmls := range opc.NH.PushedMPLSLabelStack {
			nheOpts = append(nheOpts,
				api.PushedMplsLabelStack(pmls.Type, uint64(pmls.Label)),
			)
		}

		// create NH Entry Option
		opts = append(opts, api.NHEntry(nheOpts...))
	case opc.NHG != nil:
		nhgeOpts := []api.GRIBIOption{
			api.ID(opc.NHG.ID),
		}
		if opc.NHG.BackupNHG != nil {
			nhgeOpts = append(nhgeOpts, api.BackupNextHopGroup(*opc.NHG.BackupNHG))
		}
		if opc.NHG.Color != nil {
			nhgeOpts = append(nhgeOpts, api.Color(*opc.NHG.Color))
		}
		if opc.NHG.ProgrammedID != nil {
			nhgeOpts = append(nhgeOpts, api.ProgrammedIndex(*opc.NHG.ProgrammedID))
		}
		for _, nh := range opc.NHG.NextHop {
			nhgeOpts = append(nhgeOpts, api.NHGNextHop(nh.Index, nh.Weight))
		}
		// create NHG Entry Option
		opts = append(opts, api.NHGEntry(nhgeOpts...))
	}
	return api.NewAFTOperation(opts...)
}
