import { WASI, Fd, File, PreopenDirectory } from "@bjorn3/browser_wasi_shim";

// TYPES ---------------------------------------------------------------

class XtermStdio extends Fd {
    constructor() {
        super();
    }
    fd_write(view8/*: Uint8Array*/, iovs/*: [wasi.Iovec]*/)/*: {ret: number, nwritten: number}*/ {
        let nwritten = 0;
        // const decoder = new TextDecoder();
        for (let iovec of iovs) {
            // console.log(iovec.buf, iovec.buf_len, view8.slice(iovec.buf, iovec.buf + iovec.buf_len));
            const buffer = view8.slice(iovec.buf, iovec.buf + iovec.buf_len);
            // const msg = decoder.decode(buffer);
            // console.log('worker: XtermStdio.fd_write msg:', msg);
            self.postMessage({ 'type': MSG_TERM_PRINT, 'payload': buffer });
            nwritten += iovec.buf_len;
        }
        return { ret: 0, nwritten };
    }
}

// CONSTANTS ---------------------------------------------------------------

const MSG_WASM_MODULE = 'wasm-module';
const MSG_TERM_PRINT = 'terminal-print';
const MSG_TRANFORM_DONE = 'transform-done';

// FUNCTIONS ---------------------------------------------------------------

// https://wasix.org/docs/api-reference/wasi/poll_oneoff
const poll_oneoff = (in_, out, nsubscriptions, nevents) => {
    // throw "my simple: async io not supported";
    console.log('poll_oneoff in_, out, nsubscriptions, nevents', in_, out, nsubscriptions, nevents);
    return 0;
};

// https://wasix.org/docs/api-reference/wasi/sock_accept
const sock_accept = (sock, fd_flags, ro_fd, ro_addr) => {
    console.log('sock_accept sock, fd_flags, ro_fd, ro_addr', sock, fd_flags, ro_fd, ro_addr);
    return 0;
};

const processMessage = async (e) => {
    console.log('worker: processMessage start');
    try {
        const msg = e.data;
        console.log('worker: got a message:', msg);
        switch (msg.type) {
            case MSG_WASM_MODULE: {
                console.log('worker: got a wasm module:', typeof msg.payload, msg.payload);
                const { wasmModule, filename, fileContentsArr } = msg.payload;
                const encoder = new TextEncoder();
                const args = ["move2kube", "transform", "-s", filename, "--qa-skip"];
                const env = [];
                const fds = [
                    // new OpenFile(new File([])), // stdin
                    // new OpenFile(new File([])), // stdout
                    // new OpenFile(new File([])), // stderr
                    new XtermStdio(), // stdin
                    new XtermStdio(), // stdout
                    new XtermStdio(), // stderr
                    new PreopenDirectory("/", {
                        "example.c": new File(encoder.encode(`#include "a"`)),
                        "hello.rs": new File(encoder.encode(`fn main() { println!("Hello World!"); }`)),
                        "dep.json": new File(encoder.encode(`{"a": 42, "b": 12}`)),
                        [filename]: new File(fileContentsArr),
                    }),
                ];
                const wasi = new WASI(args, env, fds, { debug: false });
                const importObject = {
                    "wasi_snapshot_preview1": wasi.wasiImport,
                };
                importObject.wasi_snapshot_preview1['poll_oneoff'] = poll_oneoff;
                importObject.wasi_snapshot_preview1['sock_accept'] = sock_accept;
                console.log('worker: importObject.wasi_snapshot_preview1', importObject.wasi_snapshot_preview1);
                const wasmModuleInstance = await WebAssembly.instantiate(wasmModule, importObject);
                console.log('worker: wasmModuleInstance', wasmModuleInstance);
                console.log('worker: wasmModuleInstance.exports', wasmModuleInstance.exports);
                console.log('worker: wasmModuleInstance.exports.memory.buffer', wasmModuleInstance.exports.memory.buffer);
                try {
                    // wasi.start(wasmModule.instance);
                    wasi.start(wasmModuleInstance);
                    // TODO: unreachable?
                    self.postMessage({ 'type': MSG_TRANFORM_DONE, 'payload': 'transformation result (no exit code)' });
                } catch (e) {
                    // console.log(typeof e);
                    // console.log(e.exit_code);
                    // console.log(Object.entries(e));
                    console.log('worker: the wasm module finished with exit code:', e);
                    // TODO: assuming the output file name is myproject.zip
                    const myprojectzip = fds[3].dir.contents["myproject.zip"].data.buffer;
                    self.postMessage({ 'type': MSG_TRANFORM_DONE, 'payload': myprojectzip });
                }
                break;
            }
            default: {
                console.error('worker: unknown message type:', msg);
            }
        }
    } catch (e) {
        console.error('worker: failed to process the message. error:', e);
    }
    console.log('worker: processMessage end');
};

const main = () => {
    console.log('worker: main start');
    self.addEventListener('message', processMessage);
    console.log('worker: main end');
};

main();
