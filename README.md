# ***Development has moved to [verrazzano/verrazzano](https://github.com/verrazzano/verrazzano)***

[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano-operator)](https://goreportcard.com/report/github.com/verrazzano/verrazzano-operator)

# Verrazzano Operator

The Verrazzano Operator is the Kubernetes operator that runs in the Verrazzano management cluster,
watches local custom resource definitions for models and bindings, and launches micro-operators
into Verrazzano managed clusters.

## Artifacts

Upon a successful release (which occurs on a Git tag), this repo publishes a Docker image: `ghcr.io/verrazzano/verrazzano-operator:tag`

## Building

To build the operator:

* Go build:

    ```
    make go-install
    ```

* Docker build:
    ```
    export ACCESS_USERNAME=<username with read access to github verrazzano project>
    export ACCESS_PASSWORD=<password for account with read access to github verrazzano project>
    make build
    ```

## Updating dependencies

The dependencies used by the Verrazzano Operator are managed by the `go.mod` file.
Any license changes related to the new dependency versions should be carefully evaluated as discussed in the [Introducing a new dependency](#introducing-a-new-dependency) section.
The steps to include changes made to dependencies are:

* Run `go get` to update references to the dependency in the `go.mod` file.

  There are three options for doing this:

  * Update to a specific tagged version of the dependency.  For example:
    ```
    go get github.com/verrazzano/verrazzano-monitoring-operator@v0.0.25
    ```
  * Update to the latest commit on a specific branch of the dependency.  For example:
    ```
    go get github.com/verrazzano/verrazzano-monitoring-operator@master
    ```
  * Update to a specific commit of the dependency.
    ```
    go get github.com/verrazzano/verrazzano-monitoring-operator@9756ffdc2472
    ```
* Ensure all `go.mod` lines referencing `github.com/verrazzano` packages use [canonical pseudo versions](https://golang.org/ref/mod#glos-pseudo-version) beginning with `v0.0.0`.
  If required, manually change the version to `v0.0.0` and retain the timestamp and commit hash.
* Run the make target:
    ```
    make go-install
    ```

## Development

### Running Locally

While developing, it's typically most efficient to run the Verrazzano Operator as an out-of-cluster process, pointing
it to your Kubernetes cluster:

```
export KUBECONFIG=<your_kubeconfig>
make go-run
```

### Running Tests

To run the integration tests, you must have [Ginkgo](https://github.com/onsi/ginkgo) and 
[Gomega](https://onsi.github.io/gomega/) installed, as follows:

```bash
$ go get github.com/onsi/ginkgo/ginkgo
$ go get github.com/onsi/gomega/...
```

To run unit tests:
```
make unit-test
```

To run integration tests:
```
make integ-test
```

## Contributing to Verrazzano

Oracle welcomes contributions to this project from anyone.  Contributions may be reporting an issue with the operator or submitting a pull request.  Before embarking on significant development that may result in a large pull request, it is recommended that you create an issue and discuss the proposed changes with the existing developers first.

If you want to submit a pull request to fix a bug or enhance an existing feature, please first open an issue and link to that issue when you submit your pull request.

If you have any questions about a possible submission, feel free to open an issue too.

## Contributing to the Verrazzano Operator repository

Pull requests can be made under The Oracle Contributor Agreement (OCA), which is available at [https://www.oracle.com/technetwork/community/oca-486395.html](https://www.oracle.com/technetwork/community/oca-486395.html).

For pull requests to be accepted, the bottom of the commit message must have the following line, using the contributor’s name and e-mail address as it appears in the OCA Signatories list.

```
Signed-off-by: Your Name <you@example.org>
```

This can be automatically added to pull requests by committing with:

```
git commit --signoff
```

Only pull requests from committers that can be verified as having signed the OCA can be accepted.

## Pull request process

*	Fork the repository.
*	Create a branch in your fork to implement the changes. We recommend using the issue number as part of your branch name, for example, `1234-fixes`.
*	Ensure that any documentation is updated with the changes that are required by your fix.
*	Ensure that any samples are updated if the base image has been changed.
*	Submit the pull request. Do not leave the pull request blank. Explain exactly what your changes are meant to do and provide simple steps on how to validate your changes. Ensure that you reference the issue you created as well. We will assign the pull request to 2-3 people for review before it is merged.

## Introducing a new dependency

Please be aware that pull requests that seek to introduce a new dependency will be subject to additional review.  In general, contributors should avoid dependencies with incompatible licenses, and should try to use recent versions of dependencies.  Standard security vulnerability checklists will be consulted before accepting a new dependency.  Dependencies on closed-source code, including WebLogic Server, will most likely be rejected.
