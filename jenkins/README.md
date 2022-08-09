### Jenkins Docker in Docker solution for parallel runs of e2e tests

Build the docker in docker image to be used as a cloud agent in Jenkins as follows:

` docker build . --tag <image_name: tag> --build-arg KUBERNETES_VERSION=v1.24.0 `

When creating / configuring the cloud nodes, under the docker Agent Template section -> Container settings: provide the volume mounts. Here, we'd mount the host ssh key path for convenience.
`/root/.ssh/:/root/.ssh`

Provide the necessary label, such that the job to run e2e tests would run in the docker container.
