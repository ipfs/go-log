package log

import (
	"os"
)

func ExampleLogger() {
	cleanup()
	os.Setenv("GOLOG_LOG_CONFIG", "example_config.json")
	SetupLogging()
	defer cleanup()

	log := Logger("parent")
	log.Info("test log from parent")

	// Output:
	// {"level":"info","logger":"parent","message":"test log from parent"}
}

func ExampleZapEventLogger_SetFieldsOnLogger() {
	cleanup()
	os.Setenv("GOLOG_LOG_CONFIG", "example_config.json")
	SetupLogging()
	defer cleanup()

	log := Logger("parent")
	log.Info("test log from parent without hostname")
	log.SetFieldsOnLogger("hostname", "host-1")
	log.Info("test log from parent")

	childlog := log.Named("child")
	childlog.Info("test log from child")
	// Output:
	// {"level":"info","logger":"parent","message":"test log from parent without hostname"}
	// {"level":"info","logger":"parent","message":"test log from parent","hostname":"host-1"}
	// {"level":"info","logger":"parent.child","message":"test log from child","hostname":"host-1"}
}

func ExampleSetFieldsOnAllLoggers() {
	cleanup()
	os.Setenv("GOLOG_LOG_CONFIG", "example_config.json")
	SetupLogging()
	defer cleanup()

	log1 := Logger("sys1")
	log2 := Logger("sys2")

	SetFieldsOnAllLoggers("hostname", "host-1", "other", "fields")
	// Any further calls to SetFieldsOnAllLoggers will be ignored
	SetFieldsOnAllLoggers("ignored", "true")

	log1.Info("test log from sys1")
	log2.Info("test log from sys2")
	// Output:
	// {"level":"info","logger":"sys1","message":"test log from sys1","hostname":"host-1","other":"fields"}
	// {"level":"info","logger":"sys2","message":"test log from sys2","hostname":"host-1","other":"fields"}
}
