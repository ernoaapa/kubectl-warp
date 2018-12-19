# kubectl warp
kubectl (Kubernetes CLI) plugin to syncronize local files to _Pod_ and executing arbitary command.

## Use cases
This can be used for example to build and run your local project in Kubernetes while using your prefed editor locally.

## Install
## MacOS with Brew
```shell
brew install ernoaapa/kubectl-plugins/warp
```
## Linux / MacOS without Brew
1. Download binary from [releases](https://github.com/ernoaapa/kubectl-warp/releases)
2. Add it to your `PATH`

## Usage
When the plugin binary is found from `PATH` you can just execute it through `kubectl`
```shell
kubectl warp --help
```

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
