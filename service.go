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
)

type (
	Service struct {
		mutex sync.Mutex
		name  string
		image string
		setup func(svc *Service) error
		wait  func(svc *Service) error
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
		setup func(svc *Service) error
	}

	CriuOption struct{}

	WaitOption struct {
		wait func(svc *Service) error
	}
)

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

func (svc *Service) start() error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	if svc.useCriu && svc.ctrd.checkpoint != nil {
		return svc.startFromCheckpoint()
	}

	return svc.startFromScratch()
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

func (svc *Service) startFromScratch() error {
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
		task, err := svc.ctrd.container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
		if err != nil {
			return errors.Wrap(err, "couldn't create task")
		}
		svc.ctrd.task = task
	}

	if err := svc.ctrd.task.Start(ctx); err != nil {
		return errors.Wrap(err, "couldn't start task")
	}

	if svc.wait != nil {
		if err := svc.wait(svc); err != nil {
			return errors.Wrap(err, "wait function failed")
		}
	}

	if svc.setup != nil {
		if err := svc.setup(svc); err != nil {
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

func (svc *Service) stop() error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

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

func (svc *Service) startFromCheckpoint() error {
	if svc.ctrd.checkpoint == nil {
		return fmt.Errorf("no checkpoint found")
	}

	if svc.ctrd.container == nil {
		return fmt.Errorf("no container found")
	}

	if !svc.useCriu {
		return fmt.Errorf("criu deactivated for this service")
	}

	task, err := svc.ctrd.container.NewTask(ctx,
		cio.NewCreator(cio.WithStdio),
		containerd.WithTaskCheckpoint(svc.ctrd.checkpoint))
	if err != nil {
		return errors.Wrap(err, "couldn't create task from checkpoint")
	}
	svc.ctrd.task = task

	if err := task.Start(ctx); err != nil {
		return errors.Wrap(err, "couldn't start task")
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
	svc.setup = o.setup
}

func WithSetup(setup func(svc *Service) error) SetupOption {
	return SetupOption{setup: setup}
}

func (o CriuOption) Apply(svc *Service) {
	svc.useCriu = true
}

func WithCriu() CriuOption {
	return CriuOption{}
}

func (o WaitOption) Apply(svc *Service) {
	svc.wait = o.wait
}

func WithWait(wait func(svc *Service) error) WaitOption {
	return WaitOption{wait: wait}
}
