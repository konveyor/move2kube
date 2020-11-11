# Move2Kube

Move2Kube is a command-line tool that accelerates the process of re-platforming to Kubernetes/Openshift. It does so by analysing the environment and source artifacts, and asking guidance from the user when required.

## Setup

1. Ensure that the move2kube executable is in path. `export PATH=$PATH:$PWD`
1. (Optional) To install dependencies such as `pack`, `kubectl` and `operator-sdk`, invoke `source installdeps.sh`.

## Usage

### One step Simple approach

`move2kube translate -s src`

### Two step involved approach

1. _Plan_ : Place source code in a directory say `src` and generate a plan. For example, you can use the `samples` directory.
    `move2kube plan -s src`
1. _Translate_ : In the same directory, invoke the below command.
    `move2kube translate`

Note: If information about any runtime instance say cloud foundry or kubernetes cluster needs to be collected use `move2kube collect`. You can place the collected data in the `src` directory used in the plan.

## Contact

For any questions reach out to us on any of the communication channels given on our website https://konveyor.github.io/move2kube/
