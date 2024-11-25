# iecs

[![CI](https://github.com/sestrella/iecs/actions/workflows/main.yml/badge.svg)](https://github.com/sestrella/iecs/actions/workflows/main.yml)

An interactive CLI for ECS to help with troubleshooting tasks like:

- Run remote commands on a container.
- Check the logs of a running container.

Compared to the AWS CLI, if no parameters are provided to the available
commands, the user would be requested to choose the desired resource from a
list of all tasks running on ECS.

## Installation

<details>
<summary>Nix users</summary>
</details>

<details>
<summary>Non-Nix users</summary>

Clone the repository:

```
git clone https://github.com/sestrella/iecs.git
```

Download and [install](https://go.dev/dl/) the appropriate Go version. Check
the version constraint on the [go.mod](go.mod) to determine which version to
use.

Compile and generate the binary:

```
go build
```

Copy the binary to a directory in the `PATH`, like `~/.local/bin`:

```
cp iecs ~/.local/bin/iecs
```

> [!NOTE]
> Check that the path where the binary is copied exists in the `PATH`
> environment variable.

</details>

## References

- https://aws.github.io/aws-sdk-go-v2/docs/getting-started/
- https://github.com/golang-standards/project-layout
