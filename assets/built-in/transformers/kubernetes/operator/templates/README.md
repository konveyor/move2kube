# Operators for various micro-services

This directory contains optional operators you can install to start different services.
Example: A MongoDB operator can start and manage multiple MongoDB instances.

## Prerequisites

1. Install the `operator-sdk` CLI tool. See https://sdk.operatorframework.io/docs/installation/

## Steps

1. `operator-sdk olm install` will install Operator Lifecycle Manager in your cluster (Make sure you are logged in first).
1. Run `kubectl apply -f .` from inside this directory. This will apply all the Kubernetes YAMLs that it finds.
1. `kubectl get csv --all-namespaces` to view the ClusterServiceVersions of all the operators that got installed on your cluster.
