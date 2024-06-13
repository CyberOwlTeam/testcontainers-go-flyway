package flyway_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/CyberOwlTeam/flyway"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
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
	defaultPostgresDbVersion  = "16.3"
	defaultPostgresPort       = "5432"
	defaultPostgresDbName     = "test_db"
	defaultPostgresDbUsername = "postgres"
	defaultPostgresDbPassword = "postgres"
)

type intPostgresContainer struct {
	*tcpostgres.PostgresContainer
}

func (c *intPostgresContainer) getInternalUrl(ctx context.Context) (string, error) {
	json, err := c.Inspect(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("jdbc:postgresql:/%s:%s/%s", json.Name, defaultPostgresPort, defaultPostgresDbName), nil
}

func (c *intPostgresContainer) getExternalUrl(ctx context.Context) (string, error) {
	url, err := c.ConnectionString(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%ssslmode=disable", url), nil // disable ssl
}

func TestFlyway(t *testing.T) {
	// given
	ctx := context.Background()
	networkContainer, err := createTestNetwork(ctx)
	require.NoError(t, err, "failed creating network container")
	postgresContainer, err := createTestPostgresContainer(ctx, networkContainer)
	require.NoError(t, err, "failed creating postgres container")
	postgresUrl, err := postgresContainer.getInternalUrl(ctx)
	require.NoError(t, err, "failed getting internal postgres url")

	// when
	flywayContainer, err := flyway.RunContainer(ctx,
		testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
		flyway.WithNetwork(networkContainer.Name),
		flyway.WithEnvUrl(postgresUrl),
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

func createTestNetwork(ctx context.Context) (*testcontainers.DockerNetwork, error) {
	return tcnetwork.New(ctx)
}

func createTestPostgresContainer(ctx context.Context, networkContainer *testcontainers.DockerNetwork) (*intPostgresContainer, error) {
	port := fmt.Sprintf("%s/tcp", defaultPostgresPort)

	postgresContainer, err := tcpostgres.RunContainer(ctx,
		tcnetwork.WithNetwork([]string{"db"}, networkContainer),
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
	if err != nil {
		return nil, err
	}

	return &intPostgresContainer{
		postgresContainer,
	}, nil
}

func requireQuery(t testing.TB, ctx context.Context, postgresContainer *intPostgresContainer) {
	postgresUrl, err := postgresContainer.getExternalUrl(ctx)
	require.NoError(t, err, "failed getting external postgres url")

	db, err := sql.Open("postgres", postgresUrl)
	require.NoError(t, err, "failed opening sql connection to postgres")
	defer db.Close()

	err = db.Ping()
	require.NoError(t, err)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO stuff (name) VALUES($1)", "test")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	rows, err := db.Query("SELECT id, name, created_timestamp FROM stuff")
	require.NoError(t, err, "failed querying postgres")
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var name string
		var created time.Time
		err := rows.Scan(&id, &name, &created)
		require.NoError(t, err)
	}

	err = rows.Err()
	require.NoError(t, err)
}