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
	DefaultFlywayVersion        = "10.15.0"
	defaultFlywayImagePattern   = "flyway/flyway:%s"
	defaultFlywayUser           = "test_user"
	defaultFlywayPassword       = "test_password"
	defaultFlywayDbUrl          = "test_flyway_db"
	defaultFlywayTable          = "schema_version"
	defaultFlywayMigrationsPath = "/flyway/sql"
	migrateCmd                  = "migrate"
	infoCmd                     = "info"
	flywayEnvUserKey            = "FLYWAY_USER"
	flywayEnvPasswordKey        = "FLYWAY_PASSWORD"
	flywayEnvUrlKey             = "FLYWAY_URL"
	flywayEnvGrouopKey          = "FLYWAY_GROUP"
	flywayEnvTableKey           = "FLYWAY_TABLE"
	flywayEnvConnectRetriesKey  = "FLYWAY_CONNECT_RETRIES"
	flywayEnvLocationsKey       = "FLYWAY_LOCATIONS"
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
			flywayEnvUserKey:           defaultFlywayUser,
			flywayEnvPasswordKey:       defaultFlywayPassword,
			flywayEnvUrlKey:            defaultFlywayDbUrl,
			flywayEnvGrouopKey:         "true",
			flywayEnvTableKey:          defaultFlywayTable,
			flywayEnvConnectRetriesKey: "3",
			flywayEnvLocationsKey:      fmt.Sprintf("filesystem:%s", defaultFlywayMigrationsPath),
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      fmt.Sprintf("./test%s", defaultFlywayMigrationsPath),
				ContainerFilePath: defaultFlywayMigrationsPath,
			},
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

func WithEnvUser(user string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_USER", user)
}

func WithEnvPassword(password string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_PASSWORD", password)
}

func WithEnvUrl(dbUrl string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_URL", dbUrl)
}

func WithEnvGroup(group string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("GROUP", group)
}

func WithEnvTable(table string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_TABLE", table)
}

func WithEnvConnectRetries(retries int) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_CONNECT_RETRIES", strconv.Itoa(retries))
}

func WithEnvLocations(locations string) testcontainers.CustomizeRequestOption {
	return withEnvSetting("FLYWAY_LOCATIONS", locations)
}

func withEnvSetting(key, group string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Env[key] = group
		return nil
	}
}

func WithMigrations(absHostFilePath string, containerFilePaths ...string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		containerFilePath := defaultFlywayMigrationsPath
		if len(containerFilePaths) > 0 && containerFilePaths[0] != "" {
			containerFilePath = containerFilePaths[0]
		}
		req.Files = []testcontainers.ContainerFile{{
			HostFilePath:      absHostFilePath,
			ContainerFilePath: containerFilePath,
		}}
		req.Env["FLYWAY_LOCATIONS"] = fmt.Sprintf("filesystem:%s", containerFilePath)
		return nil
	}
}

func BuildFlywayImageVersion(version ...string) string {
	if len(version) > 0 {
		return fmt.Sprintf(defaultFlywayImagePattern, version[0])
	}
	return fmt.Sprintf(defaultFlywayImagePattern, DefaultFlywayVersion)
}
