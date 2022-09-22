package app

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/karimra/gnmic/utils"
	"github.com/karimra/gribic/api"
	"github.com/karimra/gribic/config"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gribi/v1/proto/gribi_aft"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/openconfig/gribigo/rib"
	"github.com/openconfig/ygot/proto/ywrapper"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
)

func (a *App) InitServerFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	//

	//
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) RunEServer(cmd *cobra.Command, args []string) error {
	err := a.Config.GetGNMIServer()
	if err != nil {
		return err
	}
	go a.startGnmiServer()

	a.Targets, err = a.GetTargets()
	if err != nil {
		return err
	}
	a.Logger.Debugf("targets: %v", a.Targets)

	numTargets := len(a.Targets)
	a.wg.Add(numTargets)

	for _, t := range a.Targets {
		go func(t *target) {
			defer a.wg.Done()
			err = a.initTarget(a.ctx, t)
			if err != nil {
				a.Logger.Errorf("target %s gRIBI init target failed: %v", t.Config.Name, err)
				return
			}
			ctx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			err = a.targetRIBGet(ctx, t)
			if err != nil {
				a.Logger.Errorf("target %q gRIBI Get failed: %v", t.Config.Name, err)
				return
			}
			a.Logger.Infof("target %q gRIBI Get success", t.Config.Name)
		}(t)
	}
	//
	a.wg.Wait()
	<-a.Context().Done()
	return nil
}

func (a *App) initTarget(ctx context.Context, t *target) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cfn = cancel
	// append credentials to context
	ctx = appendCredentials(ctx, t.Config)
	// create a grpc conn
CR_CLIENT:
	err := a.CreateGrpcClient(ctx, t, a.createBaseDialOpts()...)
	if err != nil {
		a.Logger.Errorf("%q failed to create a gRPC client: %v", t.Config.Name, err)
		time.Sleep(2 * time.Second)
		goto CR_CLIENT
	}
	a.Logger.Infof("%q created gRPC client", t.Config.Name)
	t.gRIBIClient = spb.NewGRIBIClient(t.conn)
	a.Logger.Infof("%q created gRIBI client", t.Config.Name)
	return nil
}

func (a *App) targetRIBGet(ctx context.Context, t *target) error {
	req, err := api.NewGetRequest(
		api.NSAll(),
		api.AFTTypeAll(),
	)
	if err != nil {
		return err
	}
	ctx = appendCredentials(ctx, t.Config)
	rsp, err := a.get(ctx, t, req)
	if err != nil {
		return err
	}
	if t.rib == nil {
		a.Logger.Infof("target %s rib is nil, creating it", t.Config.Name)
		t.rib = rib.New(t.Config.DefaultNI)
	}
	for i, afte := range rsp.GetEntry() {
		ni := afte.GetNetworkInstance()
		if ni == "" {
			a.Logger.Printf("target %q returned an AFT with an empty network instance", t.Config.Name)
			continue
		}
		if _, ok := t.rib.NetworkInstanceRIB(ni); !ok {
			err = t.rib.AddNetworkInstance(ni)
			if err != nil {
				return err
			}
		}
		var err error
		var fails []*rib.OpResult
		switch e := afte.Entry.(type) {
		case *spb.AFTEntry_Ipv4:
			_, fails, err = t.rib.AddEntry(afte.NetworkInstance, &spb.AFTOperation{
				Id:              uint64(i),
				NetworkInstance: afte.GetNetworkInstance(),
				Op:              spb.AFTOperation_ADD,
				Entry: &spb.AFTOperation_Ipv4{
					Ipv4: e.Ipv4,
				},
			})
		case *spb.AFTEntry_NextHop:
			_, fails, err = t.rib.AddEntry(afte.NetworkInstance, &spb.AFTOperation{
				Id:              uint64(i),
				NetworkInstance: afte.GetNetworkInstance(),
				Op:              spb.AFTOperation_ADD,
				Entry: &spb.AFTOperation_NextHop{
					NextHop: e.NextHop,
				},
			})
		case *spb.AFTEntry_NextHopGroup:
			// WR: need to set Wight value to make it resolvable ?
			for i, nh := range e.NextHopGroup.GetNextHopGroup().GetNextHop() {
				if nh.GetNextHop() == nil {
					e.NextHopGroup.NextHopGroup.NextHop[i] = &gribi_aft.Afts_NextHopGroup_NextHopKey{
						Index:   nh.Index,
						NextHop: &gribi_aft.Afts_NextHopGroup_NextHop{Weight: &ywrapper.UintValue{}},
					}
				}
			}
			//
			_, fails, err = t.rib.AddEntry(afte.NetworkInstance, &spb.AFTOperation{
				Id:              uint64(i),
				NetworkInstance: afte.GetNetworkInstance(),
				Op:              spb.AFTOperation_ADD,
				Entry: &spb.AFTOperation_NextHopGroup{
					NextHopGroup: e.NextHopGroup,
				},
			})
		case *spb.AFTEntry_Ipv6:
		case *spb.AFTEntry_MacEntry:
		case *spb.AFTEntry_Mpls:
		case *spb.AFTEntry_PolicyForwardingEntry:
		}
		if err != nil {
			return err
		}
		if len(fails) > 0 {
			for _, failOp := range fails {
				a.Logger.Errorf("target %q OP failed:\nindex: %v\nop:\n%s\nerror: %v", t.Config.Name, failOp.ID, prototext.Format(failOp.Op), failOp.Error)
			}
		}
	}
	return nil
}

func (a *App) startGnmiServer() {
	if a.Config.GnmiServer == nil {
		return
	}
	a.unaryRPCsem = semaphore.NewWeighted(a.Config.GnmiServer.MaxUnaryRPC)
	//
	var l net.Listener
	var err error
	network := "tcp"
	addr := a.Config.GnmiServer.Address
	if strings.HasPrefix(a.Config.GnmiServer.Address, "unix://") {
		network = "unix"
		addr = strings.TrimPrefix(addr, "unix://")
	}

	opts, err := a.gRPCServerOpts()
	if err != nil {
		a.Logger.Printf("failed to build gRPC server options: %v", err)
		return
	}
	for {
		l, err = net.Listen(network, addr)
		if err != nil {
			a.Logger.Printf("failed to start gRPC server listener: %v", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	a.grpcServer = grpc.NewServer(opts...)
	gnmi.RegisterGNMIServer(a.grpcServer, a)
	//
	ctx, cancel := context.WithCancel(a.ctx)
	go func() {
		err = a.grpcServer.Serve(l)
		if err != nil {
			a.Logger.Printf("gRPC server shutdown: %v", err)
		}
		cancel()
	}()
	for range ctx.Done() {
	}
}

func (a *App) gRPCServerOpts() ([]grpc.ServerOption, error) {
	opts := make([]grpc.ServerOption, 0)
	if a.Config.GnmiServer.EnableMetrics && a.reg != nil {
		grpcMetrics := grpc_prometheus.NewServerMetrics()
		opts = append(opts,
			grpc.StreamInterceptor(grpcMetrics.StreamServerInterceptor()),
			grpc.UnaryInterceptor(grpcMetrics.UnaryServerInterceptor()),
		)
		a.reg.MustRegister(grpcMetrics)
	}

	tlscfg, err := utils.NewTLSConfig(
		a.Config.GnmiServer.CaFile,
		a.Config.GnmiServer.CertFile,
		a.Config.GnmiServer.KeyFile,
		a.Config.GnmiServer.SkipVerify,
		true,
	)
	if err != nil {
		return nil, err
	}
	if tlscfg != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlscfg)))
	}

	return opts, nil
}

func (a *App) Capabilities(context.Context, *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Capabilities not implemented")
}

func (a *App) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}

func (a *App) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	numUpdates := len(req.GetUpdate())
	numReplaces := len(req.GetReplace())
	numDeletes := len(req.GetDelete())
	if numUpdates+numReplaces+numDeletes == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing update/replace/delete path(s)")
	}
	//
	targetName := req.GetPrefix().GetTarget()
	pr, _ := peer.FromContext(ctx)
	a.Logger.Printf("received Set request from %q to target %q", pr.Addr, targetName)
	//
	targets, err := a.selectGNMITargets(targetName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not find targets: %v", err)
	}
	numTargets := len(targets)
	if numTargets == 0 {
		return nil, status.Errorf(codes.NotFound, "unknown target(s) %q", targetName)
	}
	//
	opts := make([]api.GRIBIOption, 0, 4)
	for _, upd := range req.Update {
		val := strings.ToLower(upd.GetVal().GetStringVal())
		p := utils.GnmiPathToXPath(upd.GetPath(), false)
		fmt.Println(p, val)
		switch p {
		case "session-parameters/ack-type":
			if val == "rib-fib" {
				opts = append(opts, api.AckTypeRibFib())
			} else {
				opts = append(opts, api.AckTypeRib())
			}
		case "session-parameters/persistence":
			if val == "preserve" {
				opts = append(opts, api.PersistencePreserve())
			} else {
				opts = append(opts, api.PersistenceDelete())
			}
		case "session-parameters/redundancy":
			if val == "single-primary" {
				opts = append(opts, api.RedundancySinglePrimary())
			} else {
				opts = append(opts, api.RedundancyAllPrimary())
			}
		case "session-parameters/election-id":
			elecID, err := config.ParseUint128(val)
			if err != nil {
				return nil, err
			}
			fmt.Println(elecID)
			opts = append(opts, api.ElectionID(elecID))
		}
	}
	modReq, err := api.NewModifyRequest(opts...)
	if err != nil {
		return nil, err
	}
	wg := new(sync.WaitGroup)
	wg.Add(numTargets)
	setRsp := &gnmi.SetResponse{
		Response: []*gnmi.UpdateResult{},
	}
	errs := make([]error, 0, numTargets)
	for name, t := range targets {
		go func(name string, t *target) {
			defer wg.Done()
			a.Logger.Infof("target %q", name)
			if t.modClient == nil {
				err = t.createModifyClient(a.ctx)
				if err != nil {
					err = fmt.Errorf("target %q modify stream client create failed: %w", name, err)
					a.Logger.Error(err)
					errs = append(errs, err)
					return
				}
			}
			a.Logger.Infof("sending modify request %q to target %q", modReq, name)
			err = t.modClient.Send(modReq)
			if err != nil {
				err = fmt.Errorf("target %q send error: %w", name, err)
				a.Logger.Error(err)
				errs = append(errs, err)
				return
			}
			rsp, err := t.modClient.Recv()
			if err != nil {
				err = fmt.Errorf("target %q rcv error: %w", name, err)
				a.Logger.Error(err)
				errs = append(errs, err)
				return
			}
			a.Logger.Infof("target %q rcv success: %v", name, rsp)
			setRsp.Response = append(setRsp.Response,
				&gnmi.UpdateResult{
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{
							{
								Name: "session-parameters",
							},
						},
					},
					Op: gnmi.UpdateResult_UPDATE,
				},
			)
		}(name, t)
	}

	wg.Wait()
	if len(errs) > 0 {
		return nil, fmt.Errorf("%v", errs)
	}
	setRsp.Timestamp = time.Now().UnixNano()
	return setRsp, nil
}

func (a *App) selectGNMITargets(targetName string) (map[string]*target, error) {
	if targetName == "" || targetName == "*" {
		return a.Targets, nil
	}
	targetsNames := strings.Split(targetName, ",")
	targets := make(map[string]*target)
	a.m.RLock()
	defer a.m.RUnlock()
OUTER:
	for i := range targetsNames {
		for n, tc := range a.Targets {
			if utils.GetHost(n) == targetsNames[i] {
				targets[n] = tc
				continue OUTER
			}
		}
		return nil, status.Errorf(codes.NotFound, "target %q is not known", targetsNames[i])
	}
	return targets, nil
}
