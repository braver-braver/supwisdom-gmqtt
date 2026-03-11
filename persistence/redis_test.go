//go:build !windows
// +build !windows

package persistence

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/DrmagicE/gmqtt/config"
	queue_test "github.com/DrmagicE/gmqtt/persistence/queue/test"
	sess_test "github.com/DrmagicE/gmqtt/persistence/session/test"
	"github.com/DrmagicE/gmqtt/persistence/subscription"
	sub_test "github.com/DrmagicE/gmqtt/persistence/subscription/test"
	unack_test "github.com/DrmagicE/gmqtt/persistence/unack/test"
	"github.com/DrmagicE/gmqtt/server"
)

var redisConfig config.RedisPersistence

func init() {
	maxIdle := uint(100)
	maxActive := uint(100)
	redisConfig = config.RedisPersistence{
		Addr:        ":6379",
		Password:    "",
		Database:    0,
		MaxIdle:     &maxIdle,
		MaxActive:   &maxActive,
		IdleTimeout: 100 * time.Second,
	}
}

type RedisSuite struct {
	suite.Suite
	p        server.Persistence
	pool     *dockertest.Pool
	resource *dockertest.Resource
}

func (s *RedisSuite) SetupSuite() {
	pool, err := newContainerPool()
	if err != nil {
		s.T().Skipf("container runtime unavailable: %v", err)
	}
	s.pool = pool
	s.pool.MaxWait = 30 * time.Second

	resource, err := s.pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   "docker.io/library/redis",
		Tag:          "7-alpine",
		ExposedPorts: []string{"6379/tcp"},
	}, func(hostConfig *dc.HostConfig) {
		hostConfig.AutoRemove = true
		hostConfig.RestartPolicy = dc.RestartPolicy{Name: "no"}
	})
	if err != nil {
		s.T().Fatalf("fail to start redis container: %v", err)
	}
	s.resource = resource
	redisConfig.Addr = resource.GetHostPort("6379/tcp")

	err = s.pool.Retry(func() error {
		p, err := NewRedis(config.Config{
			Persistence: config.Persistence{
				Type:  config.PersistenceTypeRedis,
				Redis: redisConfig,
			},
		})
		if err != nil {
			return err
		}
		defer func() {
			_ = p.Close()
		}()
		return p.Open()
	})
	if err != nil {
		s.T().Fatalf("fail to open redis: %v", err)
	}
}

func (s *RedisSuite) SetupTest() {
	p, err := NewRedis(config.Config{
		Persistence: config.Persistence{
			Type:  config.PersistenceTypeRedis,
			Redis: redisConfig,
		},
	})
	if err != nil {
		s.T().Fatalf("fail to create redis persistence: %v", err)
	}
	if err := p.Open(); err != nil {
		s.T().Fatalf("fail to open redis persistence: %v", err)
	}
	s.p = p
}

func (s *RedisSuite) TearDownTest() {
	if s.p != nil {
		_ = s.p.Close()
		s.p = nil
	}
	conn, err := redigo.Dial("tcp", redisConfig.Addr)
	if err != nil {
		s.T().Fatalf("fail to connect redis: %v", err)
	}
	defer conn.Close()
	if pswd := redisConfig.Password; pswd != "" {
		if _, err := conn.Do("AUTH", pswd); err != nil {
			s.T().Fatalf("fail to auth redis: %v", err)
		}
	}
	if _, err := conn.Do("SELECT", redisConfig.Database); err != nil {
		s.T().Fatalf("fail to select redis db: %v", err)
	}
	if _, err := conn.Do("FLUSHALL"); err != nil {
		s.T().Fatalf("fail to flush redis: %v", err)
	}
}

func (s *RedisSuite) TearDownSuite() {
	if s.pool != nil && s.resource != nil {
		_ = s.pool.Purge(s.resource)
	}
}

func (s *RedisSuite) TestQueue() {
	a := assert.New(s.T())
	cfg := queue_test.TestServerConfig
	cfg.Persistence.Redis = redisConfig
	qs, err := s.p.NewQueueStore(cfg, queue_test.TestNotifier, queue_test.TestClientID)
	a.Nil(err)
	queue_test.TestQueue(s.T(), qs)
}

func (s *RedisSuite) TestSubscription() {
	newFn := func() subscription.Store {
		st, err := s.p.NewSubscriptionStore(config.Config{})
		if err != nil {
			panic(err)
		}
		return st
	}
	sub_test.TestSuite(s.T(), newFn)
}

func (s *RedisSuite) TestSession() {
	a := assert.New(s.T())
	st, err := s.p.NewSessionStore(config.Config{})
	a.Nil(err)
	sess_test.TestSuite(s.T(), st)
}

func (s *RedisSuite) TestUnack() {
	a := assert.New(s.T())
	st, err := s.p.NewUnackStore(unack_test.TestServerConfig, unack_test.TestClientID)
	a.Nil(err)
	unack_test.TestSuite(s.T(), st)
}

func TestRedis(t *testing.T) {
	suite.Run(t, &RedisSuite{})
}

func newContainerPool() (*dockertest.Pool, error) {
	var errs []string
	for _, endpoint := range containerEndpoints() {
		pool, err := newRuntimePool(endpoint)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%q: %v", endpoint, err))
			continue
		}
		return pool, nil
	}
	return nil, errors.New(strings.Join(errs, "; "))
}

func containerEndpoints() []string {
	endpoints := make([]string, 0, 4)
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		endpoints = append(endpoints, host)
	}
	for _, socket := range []string{
		fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Getuid()),
		"unix:///run/podman/podman.sock",
		"",
	} {
		if socket == "" {
			endpoints = append(endpoints, socket)
			continue
		}
		if _, err := os.Stat(strings.TrimPrefix(socket, "unix://")); err == nil {
			endpoints = append(endpoints, socket)
		}
	}
	return endpoints
}

func newRuntimePool(endpoint string) (*dockertest.Pool, error) {
	var (
		pool *dockertest.Pool
		err  error
	)
	withContainerProxyDisabled(func() {
		pool, err = dockertest.NewPool(endpoint)
		if err != nil {
			return
		}
		err = pool.Client.Ping()
	})
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func withContainerProxyDisabled(fn func()) {
	keys := []string{
		"http_proxy", "https_proxy", "all_proxy", "no_proxy",
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
	}
	saved := make(map[string]*string, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			v := value
			saved[key] = &v
		} else {
			saved[key] = nil
		}
		_ = os.Unsetenv(key)
	}
	defer func() {
		for _, key := range keys {
			if value := saved[key]; value != nil {
				_ = os.Setenv(key, *value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	}()
	fn()
}
