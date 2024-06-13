package flyway_test

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/CyberOwlTeam/flyway"

	"github.com/testcontainers/testcontainers-go"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
)

func ExampleRunContainer() {
	// runFlywayContainer {
	ctx := context.Background()
	nw, err := tcnetwork.New(ctx)
	if err != nil {
		log.Fatalf("failed to start network: %s", err) // nolint:gocritic
	}
	postgresContainer, err := createTestPostgresContainer(ctx, nw)
	if err != nil {
		log.Fatalf("failed to start postgres container: %s", err) // nolint:gocritic
	}

	flywayContainer, err := flyway.RunContainer(ctx,
		testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
		tcnetwork.WithNetwork([]string{"flyway"}, nw),
		flyway.WithEnvUrl(postgresContainer.getNetworkUrl()),
		flyway.WithEnvUser(defaultPostgresDbUsername),
		flyway.WithEnvPassword(defaultPostgresDbPassword),
		flyway.WithMigrations(filepath.Join("testdata", flyway.DefaultFlywayMigrationsPath)),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err) // nolint:gocritic
	}

	// Clean up the container
	defer func() {
		if err := flywayContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err) // nolint:gocritic
		}
	}()
	//}

	state, err := flywayContainer.State(ctx)
	if err != nil {
		log.Fatalf("failed to get container state: %s", err) // nolint:gocritic
	}

	fmt.Println(state.Running)  // the container should terminate immediately
	fmt.Println(state.ExitCode) // the exit code should be 0, as the flyway migrations were successful

	// Output:
	// false
	// 0
}
