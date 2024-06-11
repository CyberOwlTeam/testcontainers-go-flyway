package flyway

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"strconv"
	"time"
)

const (
	DefaultFlywayVersion        = "10.10.0"
	defaultFlywayImagePattern   = "flyway/flyway:%s"
	defaultFlywayContainerName  = "test_flyway_container"
	defaultNetworkContainerName = "test_network_container"
	defaultFlywayUser           = "test_user"
	defaultFlywayPassword       = "test_password"
	defaultFlywayDbUrl          = "test_flyway"
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
		Name:       defaultFlywayContainerName,
		Image:      BuildFlywayImageVersion(DefaultFlywayVersion),
		WaitingFor: wait.ForExit().WithExitTimeout(30 * time.Second),
		Networks:   []string{defaultNetworkContainerName},
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

	return &FlywayContainer{Container: container}, nil
}

func WithEnvUser(user string) testcontainers.CustomizeRequestOption {
	return WithEnv("FLYWAY_USER", user)
}

func WithEnvPassword(password string) testcontainers.CustomizeRequestOption {
	return WithEnv("FLYWAY_PASSWORD", password)
}

func WithEnvUrl(dbUrl string) testcontainers.CustomizeRequestOption {
	return WithEnv("FLYWAY_URL", dbUrl)
}

func WithEnvGroup(group string) testcontainers.CustomizeRequestOption {
	return WithEnv("GROUP", group)
}

func WithEnvTable(table string) testcontainers.CustomizeRequestOption {
	return WithEnv("FLYWAY_TABLE", table)
}

func WithEnvConnectRetries(retries int) testcontainers.CustomizeRequestOption {
	return WithEnv("FLYWAY_CONNECT_RETRIES", strconv.Itoa(retries))
}

func WithEnvLocations(locations string) testcontainers.CustomizeRequestOption {
	return WithEnv("FLYWAY_LOCATIONS", locations)
}

func WithEnv(key, group string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Env[key] = group
		return nil
	}
}

func WithNetwork(network string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Networks = append(req.Networks, network)
		return nil
	}
}

func WithLocations(absHostFilePath string, containerFilePaths ...string) testcontainers.CustomizeRequestOption {
	return WithMigrations(absHostFilePath, containerFilePaths...)
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

func BuildFlywayImageVersion(version string) string {
	return fmt.Sprintf(defaultFlywayImagePattern, version)
}
