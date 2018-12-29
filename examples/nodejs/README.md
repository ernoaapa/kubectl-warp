# kubectl warp NodeJS example
This is an example of using `kubectl warp` for developing NodeJS app "real-time" in Kubernetes cluster.

`kubectl warp` start temporary _Pod_ and keep your local files insync so that you can make changes to files locally, but run the actual code in the Kubernetes cluster.

## Prerequisites
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Install [kubectl warp](https://github.com/ernoaapa/kubectl-warp#install)

Here we
1. start official `node` Docker image in Kubernetes
2. sync current directory to the container
3. run `npm install` 
4. start watching changes with `npm run watch` command

```shell
kubectl warp -i -t --image node nodejs-example -- npm install && npm run watch
```
