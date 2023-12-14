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
$ cd m2k-web-ui/ && pnpm install
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

```
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

To publish to Github pages run:

```shell
$ make all
$ make copy-web
```

This copies the built `m2k-web-ui/dist` directory into `docs` directory.
