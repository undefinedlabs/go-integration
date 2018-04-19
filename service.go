package integration

import (
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type (
	Service struct {
		mutex sync.Mutex
		name  string
		image string
		setup SetupOption
		wait  WaitOption
		ctrd  struct {
			image      containerd.Image
			container  containerd.Container
			task       containerd.Task
			checkpoint containerd.Image
		}
		useCriu bool
	}

	ServiceOption interface {
		Apply(*Service)
	}

	SetupOption struct {
		f func(svc *Service) error
	}

	CriuOption struct{}

	WaitOption struct {
		f       func(svc *Service) error
		timeout time.Duration
	}
)

const defaultWaitTimeout = 10 * time.Second

func NewService(name string, image string, opts ...ServiceOption) *Service {
	svc := &Service{name: name, image: image}
	for _, opt := range opts {
		opt.Apply(svc)
	}
	if err := createGlobalClient(); err == nil {
		svc.pull()
	}
	return svc
}

func (svc *Service) Start() error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	return svc.start()
}

func (svc *Service) pull() error {
	if svc.ctrd.image == nil {
		ref := svc.image
		r, err := reference.ParseNormalizedNamed(ref)
		if err == nil {
			ref = reference.TagNameOnly(r).String()
		}

		image, err := client.Pull(ctx, ref, containerd.WithPullUnpack)
		if err != nil {
			return errors.Wrap(err, "couldn't pull image")
		}
		svc.ctrd.image = image
	}
	return nil
}

func (svc *Service) start() (err error) {
	defer func() {
		if err != nil {
			svc.stop()
		}
	}()

	if err := svc.pull(); err != nil {
		return err
	}

	if svc.ctrd.container == nil {
		container, err := client.NewContainer(ctx, svc.name,
			containerd.WithNewSnapshot(fmt.Sprintf("%s-snapshot", svc.name), svc.ctrd.image),
			containerd.WithNewSpec(
				oci.WithImageConfig(svc.ctrd.image),
				oci.WithHostNamespace(specs.NetworkNamespace),
				oci.WithHostHostsFile,
				oci.WithHostResolvconf,
			),
		)
		if err != nil {
			return errors.Wrap(err, "couldn't create container")
		}
		svc.ctrd.container = container
	}

	if svc.ctrd.task == nil {
		var opts []containerd.NewTaskOpts
		if svc.ctrd.checkpoint != nil {
			opts = append(opts, containerd.WithTaskCheckpoint(svc.ctrd.checkpoint))
		}
		task, err := svc.ctrd.container.NewTask(ctx, cio.NewCreator(cio.WithStdio), opts...)
		if err != nil {
			return errors.Wrap(err, "couldn't create task")
		}
		svc.ctrd.task = task
	}

	if err := svc.ctrd.task.Start(ctx); err != nil {
		return errors.Wrap(err, "couldn't start task")
	}

	if svc.ctrd.checkpoint == nil && svc.wait.f != nil {
		c := make(chan error, 1)
		go func() {
			c <- svc.wait.f(svc)
		}()
		select {
		case err := <-c:
			if err != nil {
				return errors.Wrap(err, "wait function failed")
			}
		case <-time.After(svc.wait.timeout):
			return fmt.Errorf("timeout waiting for service to start")
		}
	}

	if svc.ctrd.checkpoint == nil && svc.setup.f != nil {
		if err := svc.setup.f(svc); err != nil {
			return errors.Wrap(err, "setup function failed")
		}
	}

	if svc.useCriu {
		image, err := svc.ctrd.task.Checkpoint(ctx)
		if err != nil {
			return err
		}
		svc.ctrd.checkpoint = image
	}

	return nil
}

func (svc *Service) Stop() error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	return svc.stop()
}

func (svc *Service) stop() error {
	isRunning, err := svc.isRunning()
	if err != nil {
		return err
	}
	if !isRunning {
		return nil
	}

	if _, err = svc.ctrd.task.Delete(ctx, containerd.WithProcessKill); err != nil {
		return err
	}
	svc.ctrd.task = nil

	if !svc.useCriu {
		if err = svc.ctrd.container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
			return err
		}
		svc.ctrd.container = nil
	}

	return nil
}

func (svc *Service) isRunning() (bool, error) {
	if svc.ctrd.task == nil {
		return false, nil
	}

	status, err := svc.ctrd.task.Status(ctx)
	if err != nil {
		return false, err
	}

	return status.Status == containerd.Running, nil
}

func (svc *Service) Hostname() string {
	return "localhost"
}

func (o SetupOption) Apply(svc *Service) {
	svc.setup = o
}

func WithSetup(setup func(svc *Service) error) SetupOption {
	return SetupOption{f: setup}
}

func (o CriuOption) Apply(svc *Service) {
	svc.useCriu = true
}

func WithCriu() CriuOption {
	return CriuOption{}
}

func (o WaitOption) Apply(svc *Service) {
	svc.wait = o
}

func WithWait(wait func(svc *Service) error, timeout time.Duration) WaitOption {
	if timeout == 0 {
		timeout = defaultWaitTimeout
	}
	return WaitOption{f: wait, timeout: timeout}
}
