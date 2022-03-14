package app

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/karimra/gnmic/utils"
	"github.com/karimra/gribic/api"
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
	"google.golang.org/grpc/metadata"
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

	targets, err := a.GetTargets()
	if err != nil {
		return err
	}
	a.Logger.Debugf("targets: %v", targets)

	numTargets := len(targets)
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
		CR_CLIENT:
			err = a.CreateGrpcClient(ctx, t, a.createBaseDialOpts()...)
			if err != nil {
				a.Logger.Printf("%q failed to create a gRPC client: %v", t.Config.Name, err)
				time.Sleep(2 * time.Second)
				goto CR_CLIENT
			}
			defer t.Close()
			t.gRIBIClient = spb.NewGRIBIClient(t.conn)
			err = a.targetRIBGet(ctx, t)
			if err != nil {
				a.Logger.Errorf("target %s gRIBI Get failed: %v", t.Config.Name, err)
			}
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					err = a.targetRIBGet(ctx, t)
					if err != nil {
						a.Logger.Errorf("target %s gRIBI Get failed: %v", t.Config.Name, err)
					}
				}
			}
		}(t)
	}
	//
	a.wg.Wait()
	<-a.Context().Done()
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

func (a *App) Set(context.Context, *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Set not implemented")
}
