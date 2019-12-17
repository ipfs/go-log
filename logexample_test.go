package log_test

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
)

func ExampleLogger() {
	logging.Cleanup()
	os.Setenv("GOLOG_LOG_CONFIG", "example_config.json")
	logging.SetupLogging()
	defer logging.Cleanup()

	log := logging.Logger("parent")
	log.Info("test log from parent")

	// Output:
	// {"level":"info","logger":"parent","message":"test log from parent"}
}

func ExampleZapEventLogger_SetFieldsOnLogger() {
	logging.Cleanup()
	os.Setenv("GOLOG_LOG_CONFIG", "example_config.json")
	logging.SetupLogging()
	defer logging.Cleanup()

	log := logging.Logger("parent")
	log.Info("test log from parent without hostname")
	log.SetFieldsOnLogger("parent", "hostname", "host-1")
	log.Info("test log from parent")

	childlog := log.Named("child")
	childlog.Info("test log from child")
	// Output:
	// {"level":"info","logger":"parent","message":"test log from parent without hostname"}
	// {"level":"info","logger":"parent","message":"test log from parent","hostname":"host-1"}
	// {"level":"info","logger":"parent.child","message":"test log from child","hostname":"host-1"}
}

func ExampleSetFieldsOnAllLoggers() {
	logging.Cleanup()
	os.Setenv("GOLOG_LOG_CONFIG", "example_config.json")
	logging.SetupLogging()
	defer logging.Cleanup()

	logging.SetFieldsOnAllLoggers("hostname", "host-1")

	log := logging.Logger("parent")
	log.Info("test log from parent")
	childlog := log.Named("child")
	childlog.Info("test log from child")
	// Output:
	// {"level":"info","logger":"parent","message":"test log from parent","hostname":"host-1"}
	// {"level":"info","logger":"parent.child","message":"test log from child","hostname":"host-1"}
}
