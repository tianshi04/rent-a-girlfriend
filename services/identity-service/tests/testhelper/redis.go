package testhelper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartRedisContainer khởi động một Redis ephemeral container
// và trả về URL kết nối (redis://host:port).
// Container sẽ tự bị destroy khi test kết thúc.
func StartRedisContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
			wait.ForListeningPort("6379/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start redis container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate redis container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	return fmt.Sprintf("redis://%s:%s", host, port.Port())
}
