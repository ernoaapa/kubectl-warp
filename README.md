<p align="center"><img src="https://github.com/ernoaapa/kubectl-warp/blob/master/media/logo.png"></p>

---
kubectl (Kubernetes CLI) plugin which is like `kubectl run` with `rsync`.

It creates temporary _Pod_ and synchronises your local files to the desired container and executes any command.

### Why
Sometimes you need to develop/execute your code **in** Kubernetes, because access to database, insufficient resources locally, need access to some specific device, use specific architecture, etc. The full build image, push, deploy cycle is way too slow for real development.

### Use cases
This can be used for example to build and run your local project in Kubernetes where's more resources, required architecture, etc. while using your prefed editor locally.

### Alternatives
- `kubectl cp` - Does full file copying, which is slow if a lot of files
- NFS - requires a lot of extra installation and configuration

### Other similar
- [telepresence](https://telepresence.io) - Executes locally and tunnels traffic from Kubernetes
- [docker-sync](https://github.com/EugenMayer/docker-sync) - Only for Docker

## How it works
`kubectl warp` is basically just combination of, simplified and modified version of `kubectl run`, `sshd-rsync` container and `kubectl port-forward` to access the container.

#### 1. Start the _Pod_
First the `warp` generates temporary SSH key pair and and starts a temporary _Pod_ with desired image and `sshd-rsync` container with the temporary public SSH public key as authorized key.

The `sshd-rsync` is just container with `sshd` daemon running in port 22 and `rsync` binary installed so the local `rsync` can sync the files to the shared volume over the SSH.
The _Pod_ have the `sshd-rsync` container defined twice, as init-container to make the initial sync before the actual container to start, and as a sidecar for the actual container to keep the files in-sync. The init-container waits one `rsync` execution and completes after succesfully so the actual containers can start.

#### 2. Open tunnel
To sync the files with `rsync` over the SSH, `warp` opens port forwarding from random local port to the _Pod_ port 22, what the `sshd-rsync` init- and sidecar-container listen.

#### 3. Initial sync
At first, the _Pod_ is in init state, and only the `sshd-rsync` is running and waiting for single sync execution. When the initial sync is done, the container completes succesfully so the _Pod_ starts the actual containers.

> The initial sync is needed so that we can start the actual container with any command. E.g. if we have shell script `test.sh` and when the container start with `./test.sh` as the command, the file must be there available before the execution.

#### 4. Continuous syncing
When the initial sync is done, the actual container start with `sshd-rsync` as a sidecar. The `warp` command continuously run `rsync` command locally to update the files in the _Pod_.

## Install

### With Krew (Kubernetes plugin manager)

Install [Krew](https://github.com/kubernetes-sigs/krew/), then run the following commands:

```shell
krew update
krew install warp
```

### MacOS with Brew
```shell
brew install rsync ernoaapa/kubectl-plugins/warp
```
### Linux / MacOS without Brew
1. Install rsync with your preferred package manager
2. Download `kubectl-warp` binary from [releases](https://github.com/ernoaapa/kubectl-warp/releases)
3. Add it to your `PATH`

## Usage
When the plugin binary is found from `PATH` you can just execute it through `kubectl` CLI
```shell
kubectl warp --help
```

### Basics
```shell
# Start bash in ubuntu image. Files in current directory will be synced to the container
kubectl warp -i -t --image ubuntu testing -- /bin/bash

# Start nodejs project in node container
cd examples/nodejs
kubectl warp -i -t --image node testing-node -- npm run watch
```

### Exclude / Include
Sometimes some directories are too unnecessary to sync so you can speed up the initial sync with
`--exclude` and `--include` flags, those gets passed to the `rsync` command, for more info see [rsync manual](http://man7.org/linux/man-pages/man1/rsync.1.html#INCLUDE/EXCLUDE_PATTERN_RULES)
```shell
# Do not sync local node_modules, because we fetch dependencies in the container as first command
cd examples/nodejs
kubectl warp -i -t --image node testing-node --exclude="node_modules/***" -- npm install && npm run watch
```

### Examples
There's some examples with different languages in [examples directory](examples/)

## Development
### Prerequisites
- Golang v1.11
- [Go mod enabled](https://github.com/golang/go/wiki/Modules)

### Build and run locally
```shell
go run ./main.go --image alpine -- ls -la

# Syncs your local files to Kubernetes and list the files
```

### Build and install locally
```shell
go install .

# Now you can use `kubectl`
kubectl warp --help
```
