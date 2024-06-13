package flyway_test

import (
	"context"
	"fmt"
	"log"

	"github.com/CyberOwlTeam/flyway"

	"github.com/testcontainers/testcontainers-go"
)

func ExampleRunContainer() {
	// runFlywayContainer {
	ctx := context.Background()
	networkContainer, err := createTestNetwork(ctx)
	if err != nil {
		log.Fatalf("failed to start network: %s", err) // nolint:gocritic
	}
	postgresContainer, err := createTestPostgresContainer(ctx, networkContainer)
	if err != nil {
		log.Fatalf("failed to start postgres container: %s", err) // nolint:gocritic
	}
	postgresUrl, err := postgresContainer.getInternalUrl(ctx)
	if err != nil {
		log.Fatalf("failed to get external postgres url: %s", err) // nolint:gocritic
	}

	flywayContainer, err := flyway.RunContainer(ctx,
		testcontainers.WithImage(flyway.BuildFlywayImageVersion()),
		flyway.WithNetwork(networkContainer.Name),
		flyway.WithEnvUrl(postgresUrl),
		flyway.WithEnvUser(defaultPostgresDbUsername),
		flyway.WithEnvPassword(defaultPostgresDbPassword),
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
