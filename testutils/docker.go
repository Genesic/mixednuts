package testutils

import (
	"fmt"
	"os"
	"time"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

type DockerInstance struct {
	Pool      *dockertest.Pool
	Container *dockertest.Resource
}

type Container interface {
	GetBoundIP(id string) string
	GetPort(id string) string
}

type TestDatabaseOptions struct {
	ContainerExpiredTime uint
	Image                string
	Tag                  string
	Env                  []string
	Init                 func(Container) error
}

func NewDockerInstance(opts TestDatabaseOptions) (*DockerInstance, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to create docker pool: %w", err)
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: opts.Image,
		Tag:        opts.Tag,
		Env:        opts.Env,
	}, func(c *docker.HostConfig) {
		c.AutoRemove = opts.ContainerExpiredTime > 0
		c.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start database container: %w", err)
	}
	_ = container.Expire(opts.ContainerExpiredTime)
	pool.MaxWait = time.Duration(opts.ContainerExpiredTime) * time.Second

	if err = pool.Retry(func() error {
		if mErr := opts.Init(container); mErr != nil {
			return mErr
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &DockerInstance{
		Pool:      pool,
		Container: container,
	}, nil
}

func (d *DockerInstance) MustClose(close func() error) {
	if err := d.Close(close); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func (d *DockerInstance) Close(close func() error) (retErr error) {
	defer func() {
		if err := d.Pool.Purge(d.Container); err != nil {
			retErr = fmt.Errorf("failed to purge database container: %w", err)
			return
		}
	}()

	if err := close(); err != nil {
		return err
	}

	return nil
}
