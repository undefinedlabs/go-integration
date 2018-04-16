package integration

import (
	"testing"
)

type (
	Test struct {
		t         *testing.T
		dependsOn []Dependency
	}

	TestOption interface {
		Apply(*Test)
	}

	Dependency struct {
		svc *Service
	}
)

func NewIntegrationTest(t *testing.T, opts ...TestOption) *Test {
	it := &Test{t: t}
	for _, o := range opts {
		o.Apply(it)
	}
	return it
}

func (it *Test) Run(name string, f func(t *testing.T)) {
	for _, dep := range it.dependsOn {
		err := dep.svc.ensureRunning()
		if err != nil {
			it.t.Fatalf("[integration] couldn't create service: %v", err)
		}
		it.t.Logf("[integration] service %s is running", dep.svc.name)
	}
	it.t.Run(name, f)
}

func (o Dependency) Apply(it *Test) {
	it.dependsOn = append(it.dependsOn, o)
}

func DependsOn(svc *Service) Dependency {
	return Dependency{svc: svc}
}
