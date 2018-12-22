# Dockwatch

A very simple library for listening to Docker container events.

## Usage

Create a watcher from a Docker client, listen to `Events` and `Errors`. Events can be `CREATE`, `UPDATE` or `DELETE`.

```go
    c, err := client.NewEnvClient()
    if err != nil {
        log.Fatal(err)
    }

    w := dockwatch.New(c)

    for {
        select {
        case e := <-w.Events:
            log.Printf("%s: %v (%s)\n", e.Type, e.Container.Names, e.Container.ID)
        case e := <-w.Errors:
            log.Println("Error:", e)
        }
    }
```

There's also a command line packaged which just demos the library by printing out events.
