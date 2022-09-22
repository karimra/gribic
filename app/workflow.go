package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	spb "github.com/openconfig/gribi/v1/proto/service"
	"google.golang.org/protobuf/proto"

	"github.com/karimra/gribic/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func (a *App) InitWorkflowFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	//
	cmd.Flags().StringVarP(&a.Config.WorkflowFile, "file", "", "", "workflow file")
	//
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(fmt.Sprintf("%s-%s", cmd.Name(), flag.Name), flag)
	})
}

func (a *App) WorkflowPreRunE(cmd *cobra.Command, args []string) error {
	if a.Config.WorkflowFile == "" {
		return errors.New("missing --file value")
	}

	err := a.Config.ReadWorkflowFile()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) WorkflowRunE(cmd *cobra.Command, args []string) error {
	targets, err := a.GetTargets()
	if err != nil {
		return err
	}
	a.Logger.Debugf("targets: %v", targets)
	numTargets := len(targets)
	a.wg.Add(numTargets)
	errCh := make(chan error, numTargets)
	for _, t := range targets {
		go func(t *target) {
			defer a.wg.Done()
			// render the workflow
			wf, err := a.Config.GenerateWorkflow(t.Config.Name)
			if err != nil {
				errCh <- fmt.Errorf("target=%q: failed to generate workflow: %v", t.Config.Name, err)
				return
			}

			// create context
			ctx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			// append credentials to context
			ctx = appendCredentials(ctx, t.Config)
			// create a gRPC conn
			err = a.CreateGrpcClient(ctx, t, a.createBaseDialOpts()...)
			if err != nil {
				errCh <- fmt.Errorf("target=%q: failed to create a GRPC client: %v", t.Config.Name, err)
				return
			}
			defer t.Close()
			//
			ex, err := a.runWorkflow(ctx, t, wf)
			if err != nil {
				a.Logger.Errorf("target=%q: failed run workflow: %v", t.Config.Name, err)
				errCh <- fmt.Errorf("target=%q: failed run workflow: %v", t.Config.Name, err)
				return
			}
			a.pm.Lock()
			fmt.Println(ex.String())
			a.pm.Unlock()
		}(t)
	}
	a.wg.Wait()
	close(errCh)

	errs := make([]error, 0) //, numTargets)
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return a.handleErrs(errs)
}

func (a *App) runWorkflow(ctx context.Context, t *target, wf *config.Workflow) (*execution, error) {
	if wf == nil {
		return nil, errors.New("nil workflow")
	}
	if len(wf.Steps) == 0 {
		return nil, fmt.Errorf("workflow %q has no steps", wf.Name)
	}

	exec := newExec(wf)

	t.gRIBIClient = spb.NewGRIBIClient(t.conn)
	// run steps
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i, s := range wf.Steps {
		if s.Name == "" {
			s.Name = fmt.Sprintf("%s.%d", wf.Name, i+1)
		}
		a.Logger.Infof("workflow=%q: target=%q: step=%s: start", wf.Name, t.Config.Name, s.Name)
		reqs, err := s.BuildRequests()
		if err != nil {
			exec.addStep(workflowStepExecution{
				Timestamp: time.Now(),
				Workflow:  wf.Name,
				Step:      s.Name,
				Target:    t.Config.Name,
				Error:     err,
			})
			return exec, err
		}
		a.Logger.Debugf("workflow=%q: target=%q: step=%s: requests: %+v", wf.Name, t.Config.Name, s.Name, reqs)
		// wait duration if any
		a.Logger.Infof("workflow=%q: target=%q: step=%s: waiting %s", wf.Name, t.Config.Name, s.Name, s.Wait)
		time.Sleep(s.Wait)
		switch rpc := strings.ToLower(s.RPC); rpc {
		case "get":
		OUTER:
			for _, req := range reqs {
				switch req := req.ProtoReflect().Interface().(type) {
				case *spb.GetRequest:
					a.Logger.Infof("workflow=%q: target=%q: step=%s: %T: %v", wf.Name, t.Config.Name, s.Name, req, req)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Request:   req,
					})
					rspCh, errCh := a.getChan(ctx, t, req)
					for {
						select {
						case <-ctx.Done():
							exec.addStep(workflowStepExecution{
								Timestamp: time.Now(),
								Workflow:  wf.Name,
								Step:      s.Name,
								Target:    t.Config.Name,
								RPC:       rpc,
								Error:     ctx.Err(),
							})
							return exec, ctx.Err()
						case rsp := <-rspCh:
							exec.addStep(workflowStepExecution{
								Timestamp: time.Now(),
								Workflow:  wf.Name,
								Step:      s.Name,
								Target:    t.Config.Name,
								RPC:       rpc,
								Response:  rsp,
							})
							a.Logger.Infof("workflow=%q: target=%q: step=%s: %T: %v", wf.Name, t.Config.Name, s.Name, rsp, rsp)
						case err := <-errCh:
							if err == io.EOF {
								continue OUTER
							}
							exec.addStep(workflowStepExecution{
								Timestamp: time.Now(),
								Workflow:  wf.Name,
								Step:      s.Name,
								Target:    t.Config.Name,
								RPC:       rpc,
								Error:     err,
							})
							return exec, err
						}
					}
				default:
					err = fmt.Errorf("workflow=%q: unexpected request type: expected GetRequest, got %T", wf.Name, req)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Error:     err,
					})
					return exec, err
				}
			}
		case "flush":
			for _, req := range reqs {
				switch req := req.ProtoReflect().Interface().(type) {
				case *spb.FlushRequest:
					a.Logger.Infof("workflow=%q: target=%q: step=%s: %T: %v", wf.Name, t.Config.Name, s.Name, req, req)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Request:   req,
					})
					rsp, err := a.flush(ctx, t, req)
					if err != nil {
						exec.addStep(workflowStepExecution{
							Timestamp: time.Now(),
							Workflow:  wf.Name,
							Step:      s.Name,
							Target:    t.Config.Name,
							RPC:       rpc,
							Error:     err,
						})
						return exec, err
					}
					a.Logger.Infof("workflow=%q: target=%q: step=%s, %T: %v\n", wf.Name, t.Config.Name, s.Name, rsp, rsp)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Response:  rsp,
					})
				default:
					err = fmt.Errorf("workflow=%q: unexpected request type: expected FlushRequest, got %T", wf.Name, req)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Error:     err,
					})
					return exec, err
				}
			}
		case "modify":
			reqCh := make(chan *spb.ModifyRequest)
			if t.modClient == nil {
				t.modClient, err = t.gRIBIClient.Modify(ctx)
				if err != nil {
					err = fmt.Errorf("failed creating modify stream: %v", err)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Error:     err,
					})
					return exec, err
				}
			}
			go func() {
				rspCh, errCh := a.modifyChan(ctx, t, reqCh)
				for {
					select {
					case <-ctx.Done():
						if ctx.Err() == nil || ctx.Err() == context.Canceled {
							return
						}
						a.Logger.Infof("workflow=%q: target=%q: step=%s: context done=%v", wf.Name, t.Config.Name, s.Name, ctx.Err())
						exec.addStep(workflowStepExecution{
							Timestamp: time.Now(),
							Workflow:  wf.Name,
							Step:      s.Name,
							Target:    t.Config.Name,
							RPC:       rpc,
							Error:     ctx.Err(),
						})
						return
					case rsp, ok := <-rspCh:
						if !ok {
							return
						}
						a.Logger.Infof("workflow=%q: target=%q: step=%s: %T: %v", wf.Name, t.Config.Name, s.Name, rsp, rsp)
						exec.addStep(workflowStepExecution{
							Timestamp: time.Now(),
							Workflow:  wf.Name,
							Step:      s.Name,
							Target:    t.Config.Name,
							RPC:       rpc,
							Response:  rsp,
						})
					case err, ok := <-errCh:
						if !ok {
							return
						}
						a.Logger.Infof("workflow=%q: target=%q: step=%s: err=%v", t.Config.Name, wf.Name, s.Name, err)
						exec.addStep(workflowStepExecution{
							Timestamp: time.Now(),
							Workflow:  wf.Name,
							Step:      s.Name,
							Target:    t.Config.Name,
							RPC:       rpc,
							Error:     err,
						})
						return
					}
				}
			}()
			for _, req := range reqs {
				switch req := req.ProtoReflect().Interface().(type) {
				case *spb.ModifyRequest:
					a.Logger.Infof("workflow=%q: target=%q: step=%s: %T: %v", wf.Name, t.Config.Name, s.Name, req, req)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Request:   req,
					})
					reqCh <- req
				default:
					err = fmt.Errorf("workflow=%q: unexpected request type: expected ModifyRequest, got %T", wf.Name, req)
					exec.addStep(workflowStepExecution{
						Timestamp: time.Now(),
						Workflow:  wf.Name,
						Step:      s.Name,
						Target:    t.Config.Name,
						RPC:       rpc,
						Error:     err,
					})
					return exec, err
				}
			}
		}
		// wait duration if any
		a.Logger.Infof("workflow=%q: target=%q: step=%s: waiting %s after execution", wf.Name, t.Config.Name, s.Name, s.WaitAfter)
		time.Sleep(s.WaitAfter)
	}
	return exec, nil
}

type workflowStepExecution struct {
	Timestamp time.Time     `json:"timestamp,omitempty"`
	Workflow  string        `json:"workflow,omitempty"`
	Step      string        `json:"step,omitempty"`
	Target    string        `json:"target,omitempty"`
	RPC       string        `json:"rpc,omitempty"`
	Request   proto.Message `json:"request,omitempty"`
	Response  proto.Message `json:"response,omitempty"`
	Error     error         `json:"error,omitempty"`
}

type execution struct {
	wf     *config.Workflow
	m      *sync.Mutex
	result []workflowStepExecution
}

func newExec(wf *config.Workflow) *execution {
	return &execution{
		wf:     wf,
		m:      &sync.Mutex{},
		result: []workflowStepExecution{},
	}
}

func (e *execution) addStep(wse workflowStepExecution) {
	e.m.Lock()
	defer e.m.Unlock()
	e.result = append(e.result, wse)
}

func (e *execution) String() string {
	b, _ := json.MarshalIndent(e.result, "", "  ")
	return string(b)
}
