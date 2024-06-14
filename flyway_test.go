package flyway_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

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
)

const (
	defaultPostgresDbVersion  = "16.3"
	defaultPostgresPort       = "5432"
	defaultPostgresSrvName    = "pgdb"
	defaultPostgresDbName     = "test_db"
	defaultPostgresDbUsername = "postgres"
	defaultPostgresDbPassword = "postgres"
)

type intPostgresContainer struct {
	*tcpostgres.PostgresContainer
}

func (c *intPostgresContainer) getNetworkUrl() string {
	return fmt.Sprintf("jdbc:postgresql://%s:%s/%s?sslmode=disable", defaultPostgresSrvName, defaultPostgresPort, defaultPostgresDbName)
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
	nw, err := tcnetwork.New(context.Background())
	require.NoError(t, err, "failed creating network")

	postgresContainer, err := createTestPostgresContainer(ctx, nw)
	require.NoError(t, err, "failed creating postgres container")

	// when
	flywayContainer, err := flyway.RunContainer(ctx,
		testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
		tcnetwork.WithNetwork([]string{"flyway"}, nw),
		flyway.WithDatabaseUrl(postgresContainer.getNetworkUrl()),
		flyway.WithUser(defaultPostgresDbUsername),
		flyway.WithPassword(defaultPostgresDbPassword),
		flyway.WithMigrations(filepath.Join("testdata", flyway.DefaultMigrationsPath)),
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

func TestFlyway_parseInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		opts []testcontainers.ContainerCustomizer
	}{
		{
			name: "missing database url",
			opts: []testcontainers.ContainerCustomizer{
				testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
				flyway.WithUser(defaultPostgresDbUsername),
				flyway.WithPassword(defaultPostgresDbPassword),
				flyway.WithMigrations(filepath.Join("testdata", flyway.DefaultMigrationsPath)),
			},
		},
		{
			name: "missing user",
			opts: []testcontainers.ContainerCustomizer{
				testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
				flyway.WithDatabaseUrl("jdbc:postgresql://localhost:5432/test_db?sslmode=disable"),
				flyway.WithPassword(defaultPostgresDbPassword),
				flyway.WithMigrations(filepath.Join("testdata", flyway.DefaultMigrationsPath)),
			},
		},
		{
			name: "missing password",
			opts: []testcontainers.ContainerCustomizer{
				testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
				flyway.WithDatabaseUrl("jdbc:postgresql://localhost:5432/test_db?sslmode=disable"),
				flyway.WithUser(defaultPostgresDbUsername),
				flyway.WithMigrations(filepath.Join("testdata", flyway.DefaultMigrationsPath)),
			},
		},
		{
			name: "missing migrations",
			opts: []testcontainers.ContainerCustomizer{
				testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
				flyway.WithDatabaseUrl("jdbc:postgresql://localhost:5432/test_db?sslmode=disable"),
				flyway.WithUser(defaultPostgresDbUsername),
				flyway.WithPassword(defaultPostgresDbPassword),
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(tt *testing.T) {
			testCase := testCase

			flywayContainer, err := flyway.RunContainer(context.Background(),
				testCase.opts...,
			)

			require.Error(tt, err, "expected error")
			require.Nil(tt, flywayContainer, "expected nil container")
		})
	}
}

func createTestPostgresContainer(ctx context.Context, nw *testcontainers.DockerNetwork) (*intPostgresContainer, error) {
	port := fmt.Sprintf("%s/tcp", defaultPostgresPort)

	postgresContainer, err := tcpostgres.RunContainer(ctx,
		tcnetwork.WithNetwork([]string{defaultPostgresSrvName}, nw),
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
	require.NoError(t, err, "failed to ping postgres")

	err = executeAsTransaction(db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO stuff (name) VALUES($1)", "test")
		return err
	})
	require.NoError(t, err, "failed to execute postgres transaction")

	rows, err := db.Query("SELECT id, name, created_timestamp FROM stuff")
	require.NoError(t, err, "failed querying postgres")
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var name string
		var created time.Time
		err := rows.Scan(&id, &name, &created)
		require.NoError(t, err, "failed to scan postgres")
	}

	err = rows.Err()
	require.NoError(t, err, "postgres error")
}

func executeAsTransaction(db *sql.DB, fUpdate func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			err = fmt.Errorf("panic occurred in transaction: %v", p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = fUpdate(tx)
	return err
}
