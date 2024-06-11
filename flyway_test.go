package flyway_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/CyberOwlTeam/flyway"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"testing"
	"time"
)

const (
	defaultPostgresDbVersion     = "13.7"
	defaultPostgresContainerName = "test_db_container"
	defaultPostgresPort          = "5432"
	defaultPostgresDbName        = "test_db"
	defaultPostgresDbUsername    = "postgres"
	defaultPostgresDbPassword    = "postgres"
)

type intPostgresContainer struct {
	*tcpostgres.PostgresContainer
}

func (c *intPostgresContainer) getInternalUrl(t testing.TB, ctx context.Context) string {
	inspect, err := c.Inspect(ctx)
	require.NoError(t, err)
	return fmt.Sprintf("jdbc:postgresql:/%s:%s/%s", inspect.Name, defaultPostgresPort, defaultPostgresDbName)
}

func (c *intPostgresContainer) getExternalUrl(t testing.TB, ctx context.Context) string {
	url, err := c.ConnectionString(ctx)
	require.NoError(t, err)
	return fmt.Sprintf("%ssslmode=disable", url) // disable ssl
}

func TestFlyway(t *testing.T) {
	// given
	ctx := context.Background()
	networkContainer := createTestNetworkContainer(t, ctx)
	postgresContainer := createTestPostgresContainer(t, ctx, networkContainer)

	// when
	flywayContainer, err := flyway.RunContainer(ctx,
		flyway.WithNetwork(networkContainer.Name),
		flyway.WithEnvUrl(postgresContainer.getInternalUrl(t, ctx)),
		flyway.WithEnvUser(defaultPostgresDbUsername),
		flyway.WithEnvPassword(defaultPostgresDbPassword),
	)
	require.NoError(t, err, "failed to run container")

	// then
	t.Cleanup(func() {
		err := flywayContainer.Terminate(ctx)
		require.NoError(t, err, "failed to terminate flyway container")

		err = postgresContainer.Terminate(ctx)
		require.NoError(t, err, "failed to terminate postgres container")
	})

	requireQuery(t, ctx, postgresContainer)

	state, err := flywayContainer.State(ctx)
	require.NoError(t, err, "failed to get container state")
	require.Emptyf(t, state.Error, "failed to get container state")
	require.Equal(t, 0, state.ExitCode, "container exit code was not as expected: migration failed")
}

func createTestNetworkContainer(t testing.TB, ctx context.Context) *testcontainers.DockerNetwork {
	networkContainer, err := tcnetwork.New(ctx)
	require.NoError(t, err)
	require.NotNil(t, networkContainer)
	return networkContainer
}

func createTestPostgresContainer(t testing.TB, ctx context.Context, networkContainer *testcontainers.DockerNetwork) *intPostgresContainer {
	posgresContainerName := defaultPostgresContainerName
	port := fmt.Sprintf("%s/tcp", defaultPostgresPort)

	postgresContainer, err := tcpostgres.RunContainer(ctx,
		withNetwork(posgresContainerName, networkContainer.Name),
		testcontainers.WithImage(fmt.Sprintf("postgres:%s", defaultPostgresDbVersion)),
		tcpostgres.WithDatabase(defaultPostgresDbName),
		tcpostgres.WithUsername(defaultPostgresDbUsername),
		tcpostgres.WithPassword(defaultPostgresDbPassword),
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.ExposedPorts = map[nat.Port]struct{}{
				nat.Port(port): {},
			}
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	require.NoError(t, err)

	return &intPostgresContainer{
		postgresContainer,
	}
}

func withNetwork(name, network string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Name = name
		req.Networks = []string{network}
		return nil
	}
}

func requireQuery(t testing.TB, ctx context.Context, postgresContainer *intPostgresContainer) {
	db, err := sql.Open("postgres", postgresContainer.getExternalUrl(t, ctx))
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	require.NoError(t, err)

	rows, err := db.Query("SELECT id, stuff_id FROM other_stuff")
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		require.NoError(t, err)
	}

	err = rows.Err()
	require.NoError(t, err)
}
