package flyway_test

import (
	"context"
	"fmt"
	"github.com/CyberOwlTeam/flyway"
	"log"
)

func ExampleRunContainer() {
	// runFlywayContainer {
	ctx := context.Background()

	flywayContainer, err := flyway.RunContainer(ctx)
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
	fmt.Println(state.ExitCode) // the exit code should be 1, as the flyway migrations could not be run

	// Output:
	// false
	// 1
}
