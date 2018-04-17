package integration

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type (
	Service struct {
		mutex      sync.Mutex
		name       string
		image      string
		setup      func(svc *Service) error
		container  containerd.Container
		task       containerd.Task
		checkpoint containerd.Image
	}

	ServiceOption interface {
		Apply(*Service)
	}

	SetupOption struct {
		setup func(svc *Service) error
	}
)

var (
	ctx        = namespaces.NamespaceFromEnv(context.Background())
	clientOpts = containerd.WithDialOpts([]grpc.DialOption{grpc.WithTimeout(time.Second * 2), grpc.WithInsecure()})
	client     *containerd.Client
)

func getClient() (*containerd.Client, error) {
	if client == nil {
		c, err := containerd.New(defaults.DefaultAddress, clientOpts)
		if err != nil {
			return nil, err
		}
		client = c
	}
	return client, nil
}

func NewService(name string, image string, opts ...ServiceOption) *Service {
	svc := &Service{name: name, image: image}
	for _, opt := range opts {
		opt.Apply(svc)
	}
	return svc
}

func (svc *Service) start() error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	if svc.checkpoint != nil {
		return svc.startFromCheckpoint()
	}
	return svc.startFromScratch()
}

func (svc *Service) startFromScratch() error {
	client, err := getClient()
	if err != nil {
		return errors.Wrap(err, "couldn't create containerd client")
	}

	ref := svc.image
	r, err := reference.ParseNormalizedNamed(ref)
	if err == nil {
		ref = reference.TagNameOnly(r).String()
	}

	image, err := client.Pull(ctx, ref, containerd.WithPullUnpack)
	if err != nil {
		return errors.Wrap(err, "couldn't pull image")
	}

	container, err := client.NewContainer(ctx, svc.name,
		containerd.WithNewSnapshot(fmt.Sprintf("%s-snapshot", svc.name), image),
		containerd.WithNewSpec(
			oci.WithImageConfig(image),
			oci.WithHostNamespace(specs.NetworkNamespace),
			oci.WithHostHostsFile,
			oci.WithHostResolvconf,
		),
	)
	if err != nil {
		return errors.Wrap(err, "couldn't create container")
	}
	svc.container = container

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return errors.Wrap(err, "couldn't create task")
	}
	svc.task = task

	if err := task.Start(ctx); err != nil {
		return errors.Wrap(err, "couldn't start task")
	}

	// TODO: properly wait until service is up
	time.Sleep(time.Second)

	if svc.setup != nil {
		if err := svc.setup(svc); err != nil {
			return errors.Wrap(err, "setup failed")
		}
	}

	image, err = svc.task.Checkpoint(ctx)
	if err != nil {
		return err
	}
	svc.checkpoint = image

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

	if _, err = svc.task.Delete(ctx, containerd.WithProcessKill); err != nil {
		return err
	}
	svc.task = nil
	return nil
}

func (svc *Service) startFromCheckpoint() error {
	if svc.checkpoint == nil {
		return fmt.Errorf("no checkpoint found")
	}

	if svc.container == nil {
		return fmt.Errorf("no container found")
	}

	task, err := svc.container.NewTask(ctx,
		cio.NewCreator(cio.WithStdio),
		containerd.WithTaskCheckpoint(svc.checkpoint))
	if err != nil {
		return errors.Wrap(err, "couldn't create task from checkpoint")
	}
	svc.task = task

	if err := task.Start(ctx); err != nil {
		return errors.Wrap(err, "couldn't start task")
	}
	return nil
}

func (svc *Service) isRunning() (bool, error) {
	if svc.task == nil {
		return false, nil
	}

	status, err := svc.task.Status(ctx)
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
