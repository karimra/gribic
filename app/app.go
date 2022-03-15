package app

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/karimra/gribic/config"
	"github.com/openconfig/gnmi/proto/gnmi"
	spb "github.com/openconfig/gribi/v1/proto/service"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/grpclog"
)

type App struct {
	ctx     context.Context
	Cfn     context.CancelFunc
	RootCmd *cobra.Command

	// wg for cmd execution
	wg *sync.WaitGroup
	// gRIBIc config
	Config *config.Config
	// gRIBI client electionID
	electionID *spb.Uint128
	// gRIBI targets, ie routers
	m       *sync.RWMutex
	Targets map[string]*target
	// gNMI server
	gnmi.UnimplementedGNMIServer
	grpcServer  *grpc.Server
	unaryRPCsem *semaphore.Weighted
	//
	Logger *log.Entry
	//
	// prometheus registry
	reg *prometheus.Registry
}

func New() *App {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.New()
	a := &App{
		ctx:     ctx,
		Cfn:     cancel,
		RootCmd: new(cobra.Command),
		Config:  config.New(),
		//
		m:       new(sync.RWMutex),
		Targets: make(map[string]*target),
		wg:      new(sync.WaitGroup),
		Logger:  log.NewEntry(logger),
	}
	return a
}

func (a *App) Context() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

func (a *App) InitGlobalFlags() {
	a.RootCmd.ResetFlags()

	a.RootCmd.PersistentFlags().StringVar(&a.Config.CfgFile, "config", "", "config file (default is $HOME/gribic.yaml)")
	a.RootCmd.PersistentFlags().StringSliceVarP(&a.Config.GlobalFlags.Address, "address", "a", []string{}, "comma separated gRIBI targets addresses")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.Username, "username", "u", "", "username")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.Password, "password", "p", "", "password")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.Port, "port", "", defaultGrpcPort, "gRPC port")
	a.RootCmd.PersistentFlags().BoolVarP(&a.Config.GlobalFlags.Insecure, "insecure", "", false, "insecure connection")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.TLSCa, "tls-ca", "", "", "tls certificate authority")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.TLSCert, "tls-cert", "", "", "tls certificate")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.TLSKey, "tls-key", "", "", "tls key")
	a.RootCmd.PersistentFlags().DurationVarP(&a.Config.GlobalFlags.Timeout, "timeout", "", 10*time.Second, "grpc timeout, valid formats: 10s, 1m30s, 1h")
	a.RootCmd.PersistentFlags().BoolVarP(&a.Config.GlobalFlags.Debug, "debug", "d", false, "debug mode")
	a.RootCmd.PersistentFlags().BoolVarP(&a.Config.GlobalFlags.SkipVerify, "skip-verify", "", false, "skip verify tls connection")
	a.RootCmd.PersistentFlags().BoolVarP(&a.Config.GlobalFlags.ProxyFromEnv, "proxy-from-env", "", false, "use proxy from environment")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.Format, "format", "", "text", "output format, one of: text, json")
	//
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.GlobalFlags.ElectionID, "election-id", "", "1:0", "gRIBI client electionID, format is high:low where both high and low are uint64")
}

func (a *App) PreRun(cmd *cobra.Command, args []string) error {
	// init logger
	a.Config.SetLogger()
	if a.Config.Debug {
		a.Logger.Logger.SetLevel(log.DebugLevel)
		grpclog.SetLogger(a.Logger) //lint:ignore SA1019 .
	}
	// a.Config.SetPersistantFlagsFromFile(a.RootCmd)
	return nil
}

func (a *App) CreateGrpcClient(ctx context.Context, t *target, opts ...grpc.DialOption) error {
	tOpts := make([]grpc.DialOption, 0, len(opts)+1)
	tOpts = append(tOpts, opts...)

	nOpts, err := t.Config.DialOpts()
	if err != nil {
		return err
	}
	tOpts = append(tOpts, nOpts...)
	timeoutCtx, cancel := context.WithTimeout(ctx, t.Config.Timeout)
	defer cancel()
	t.conn, err = grpc.DialContext(timeoutCtx, t.Config.Address, tOpts...)
	return err
}

func (a *App) createBaseDialOpts() []grpc.DialOption {
	opts := []grpc.DialOption{grpc.WithBlock()}
	if !a.Config.ProxyFromEnv {
		opts = append(opts, grpc.WithNoProxy())
	}
	if a.Config.Gzip {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	}
	return opts
}

func parseUint128(v string) (*spb.Uint128, error) {
	if v == "" {
		return &spb.Uint128{}, nil
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
