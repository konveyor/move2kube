# Dockerfile Containerization

## Description

We elaborate on how new language-platform support could be added using docker file containerization with a running example of one of the samples (namely, `java-maven`). 

## Steps

1. Follow the steps mentioned [here](https://github.com/konveyor/move2kube-demos/blob/main/tutorials/dockerfile-containerization.md) to create and test the dockerfile and script before actually including it in the code.
2. If tests pass, copy the sample you have used for the new language-platform in:
    ```
    https://github.com/konveyor/move2kube/tree/main/samples
    ```
3. Create a directory for the new language-platform in:
    ```
    https://github.com/konveyor/move2kube/tree/main/internal/assets/dockerfiles
    ```
    and add the **dockerfile template** and `m2kdfdetect.sh` to it.
4. Perform the following steps to build the code:
    ```
    make generate
    make build
    ```
5. Repeat the steps in **Generate and test** section from this [document](https://github.com/konveyor/move2kube-demos/blob/main/tutorials/dockerfile-containerization.md) to test the dockerfile and script created from previous steps.
6. Updates any related test cases for the above changes.
7. Once the test passes, commit the code with sign-off and create a pull request by following steps specified [here](https://github.com/konveyor/move2kube/blob/main/contributing.md).
