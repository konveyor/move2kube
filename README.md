# Move2Kube in WASM/WASI

A version of Move2Kube that can run in WASM/WASI.
Goal is to run in the browser using a virtual file system.
Similar to https://github.com/HarikrishnanBalagopal/test-wasi-fs-browser/tree/main

## Prerequisites

- Go v1.21 or higher to use WASI support
- NodeJS v16.9.0 or higher to use Corepack https://nodejs.org/api/corepack.html . Run `corepack enable` to allow NodeJS to install package managers like `pnpm` . You can multiple NodeJS versions using NVM https://github.com/nvm-sh/nvm
- Python3 HTTP server
- Optional: Custom fork of TinyGo with stubs added for required system calls https://github.com/Prakhar-Agarwal-byte/tinygo/tree/stub

### Install dependencies

This should only be run once (or if the Javascript dependencies change)

```shell
$ cd m2k-web-ui/ && pnpm install && cd -
```

## Usage

### Development

If only changing the Web UI, HTML, CSS, Javascript, etc.  
then you can start the WebPack development server.
From the root directory run the following command:

```shell
$ make dev
```

If the WebAssembly (Golang code) changes then you will need to rebuild the WASM module.

```shell
$ make build
```

### Production

From the root directory run the following command:

```shell
$ make all
```

This will:
- build the WASM module
- build the Javascript Webpack bundle
- start a Python HTTP server to serve the webpage

You can go to http://localhost:8080/ to access the UI

## Publish

The webapp is published to Github pages here https://move2kube.konveyor.io/experimental

First create the production build:

```shell
$ make all
```

Then replace the `assets/wasm` directory in the website repo with the `m2k-web-ui/dist` directory that was just built.

https://github.com/konveyor/move2kube-website/tree/main/assets/wasm

Then make a PR to the website repo.
