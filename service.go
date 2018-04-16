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
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"time"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type (
	Service struct {
		name  string
		image string
	}
)

const namespace = "buildkit"

var ctx = namespaces.WithNamespace(context.Background(), namespace)
var clientOpts = containerd.WithDialOpts([]grpc.DialOption{grpc.WithTimeout(time.Second * 2), grpc.WithInsecure()})

func NewService(name string, image string) *Service {
	return &Service{name: name, image: image}
}

func (svc *Service) ensureRunning() error {
	if svc.isRunning() {
		return nil
	}

	return svc.start()
}

func (svc *Service) start() error {
	client, err := containerd.New(defaults.DefaultAddress, clientOpts)
	if err != nil {
		return errors.Wrap(err, "couldn't create containerd client")
	}
	defer client.Close()

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
	// defer container.Delete(ctx, containerd.WithSnapshotCleanup)

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return errors.Wrap(err, "couldn't create task")
	}
	// defer task.Delete(ctx)

	task.Wait(ctx)

	if err := task.Start(ctx); err != nil {
		return errors.Wrap(err, "couldn't start task")
	}
	return nil
}

func (svc *Service) isRunning() bool {
	client, err := containerd.New(defaults.DefaultAddress, clientOpts)
	if err != nil {
		return false
	}
	defer client.Close()

	containers, err := client.Containers(ctx, fmt.Sprintf("name==%s", svc.name))
	if err != nil {
		return false
	}

	return len(containers) > 0
}
