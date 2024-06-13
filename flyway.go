package flyway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DefaultVersion        = "10.15.0"
	DefaultMigrationsPath = "/flyway/sql"

	defaultImagePattern = "flyway/flyway:%s"
	defaultUser         = "test_user"
	defaultPassword     = "test_password"
	defaultDbUrl        = "test_flyway_db"
	defaultTable        = "schema_version"
	migrateCmd          = "migrate"
	infoCmd             = "info"

	// flyway environment variables
	flywayEnvUserKey           = "FLYWAY_USER"
	flywayEnvPasswordKey       = "FLYWAY_PASSWORD"
	flywayEnvUrlKey            = "FLYWAY_URL"
	flywayEnvGrouopKey         = "FLYWAY_GROUP"
	flywayEnvTableKey          = "FLYWAY_TABLE"
	flywayEnvConnectRetriesKey = "FLYWAY_CONNECT_RETRIES"
	flywayEnvLocationsKey      = "FLYWAY_LOCATIONS"
)

// FlywayContainer represents the Flyway container type used in the module
type FlywayContainer struct {
	testcontainers.Container
}

// RunContainer creates an instance of the Flyway container type
func RunContainer(ctx context.Context, opts ...testcontainers.ContainerCustomizer) (*FlywayContainer, error) {
	req := testcontainers.ContainerRequest{
		WaitingFor: wait.ForExit().WithExitTimeout(30 * time.Second),
		Env: map[string]string{
			flywayEnvUserKey:           defaultUser,
			flywayEnvPasswordKey:       defaultPassword,
			flywayEnvUrlKey:            defaultDbUrl,
			flywayEnvGrouopKey:         "true",
			flywayEnvTableKey:          defaultTable,
			flywayEnvConnectRetriesKey: "3",
			flywayEnvLocationsKey:      fmt.Sprintf("filesystem:%s", DefaultMigrationsPath),
		},
		Cmd: []string{
			migrateCmd, infoCmd,
		},
	}

	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&genericContainerReq); err != nil {
			return nil, fmt.Errorf("failed to customize flyway container: %w", err)
		}
	}

	container, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	if err != nil {
		return nil, err
	}

	state, err := container.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container state: %s", err)
	} else if state.ExitCode != 0 {
		if state.Health != nil {
			return nil, fmt.Errorf("the container state is not healthy: %d/%s", state.ExitCode, state.Health.Status)
		}
		return nil, fmt.Errorf("the container state is not healthy: %d", state.ExitCode)
	}

	return &FlywayContainer{
		Container: container,
	}, nil
}

func WithUser(user string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_USER", user)
}

func WithPassword(password string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_PASSWORD", password)
}

func WithDatabaseUrl(dbUrl string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_URL", dbUrl)
}

func WithGroup(group string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("GROUP", group)
}

func WithTable(table string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_TABLE", table)
}

func WithConnectRetries(retries int) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_CONNECT_RETRIES", strconv.Itoa(retries))
}

func withEnvSetting(key, group string) testcontainers.CustomizeRequestOption {
	return testcontainers.WithEnv(map[string]string{
		key: group,
	})
}

func WithLocations(absHostFilePath string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Files = []testcontainers.ContainerFile{{
			HostFilePath:      absHostFilePath,
			ContainerFilePath: DefaultMigrationsPath,
		}}

		return withEnvSetting("FLYWAY_LOCATIONS", fmt.Sprintf("filesystem:%s", DefaultMigrationsPath))(req)
	}
}

func BuildFlywayImageVersion(version ...string) string {
	if len(version) > 0 {
		return fmt.Sprintf(defaultImagePattern, version[0])
	}
	return fmt.Sprintf(defaultImagePattern, DefaultVersion)
}
