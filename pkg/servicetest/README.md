# Package `servicetest`

This package provides basic building blocks for testing your superwatcher-based services.

The idea here is you test your services by running it against a known environment,
which in this case is JSON log files using `reorgsim` chain simulation.

> See [demotest](../../superwatcher-demo/demotest/) for usage examples.

Most of the name definitions are public, so users can have to items defined here,
such as struct `DebugEngine` and function `RunService`.

The most basic way to use servicetest is to call `RunService`
with a `superwatcher.WatcherEmitter` and `superwatcher.WatcherEngine`,
although it's better to use `TestCase` for multiple test cases.

Users should rely on this package in addition to their own unit tests,
especially if the services in question are poller-type services.