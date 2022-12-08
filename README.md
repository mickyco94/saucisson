# Saucisson

Saucisson is a declarative task runner that defines pairs of conditions and executions to be run in case of those conditions occurring.

```yaml
services:
  - name: "open top"
    condition:
      type: "process"
      config:
        executable: "top"
        state: "open"
    execute:
      type: "shell"
      config:
        command: "echo top opened"
  - name: "close top"
    condition:
      type: "process"
      config:
        executable: "top"
        state: "close"
    execute:
      type: "shell"
      config:
        shell: "bash"
        command: "echo top closed"
```

# Installation

Git:

```sh
git clone https://github.com/mickyco94/saucisson.git
cd saucisson
go build -o ~/go/bin/saucisson cmd/main.go
```

The above installation methods assumes that `~/go/bin/` is added to your `$PATH`.

# Run

Saucisson is run using the following:

Run with specified config:

```sh
saucisson -c examples/template.yml run
```

Run with default config (~/.saucisson.yml)

```sh
saucisson run
```
