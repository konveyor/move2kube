import { WASI, Fd, File, PreopenDirectory, Directory } from "@bjorn3/browser_wasi_shim";

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
const MSG_TRANFORM_ERR = 'transform-error';

const WASI_PREVIEW_1 = 'wasi_snapshot_preview1';
const OUTPUT_DIR = 'my-m2k-output';

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

let MY_DEBUG_FDS = null;

const processMessage = async (e) => {
    console.log('worker: processMessage start');
    try {
        const msg = e.data;
        console.log('worker: got a message:', msg);
        switch (msg.type) {
            case MSG_WASM_MODULE: {
                console.log('worker: got a wasm module:', typeof msg.payload, msg.payload);
                // const { wasmModule, filename, fileContentsArr, filenameCust, fileContentsCustArr } = msg.payload;
                const { wasmModule, srcFilename, srcContents, custFilename, custContents, configFilename, configContents } = msg.payload;
                // const encoder = new TextEncoder();
                const args = ["move2kube", "transform", "--qa-skip", "--source", srcFilename, "--output", OUTPUT_DIR];
                const preOpenDir = {
                    // "example.c": new File(encoder.encode(`#include "a"`)),
                    // "hello.rs": new File(encoder.encode(`fn main() { println!("Hello World!"); }`)),
                    // "dep.json": new File(encoder.encode(`{"a": 42, "b": 12}`)),
                    [srcFilename]: new File(srcContents),
                };
                if (custFilename && custContents) {
                    args.push('--customizations', custFilename);
                    preOpenDir[custFilename] = new File(custContents);
                }
                if (configFilename && configContents) {
                    args.push('--config', configFilename);
                    preOpenDir[configFilename] = new File(configContents);
                }
                const env = [];
                const fds = [
                    // new OpenFile(new File([])), // stdin
                    // new OpenFile(new File([])), // stdout
                    // new OpenFile(new File([])), // stderr
                    new XtermStdio(), // stdin
                    new XtermStdio(), // stdout
                    new XtermStdio(), // stderr
                    new PreopenDirectory("/", preOpenDir),
                ];
                MY_DEBUG_FDS = fds;
                const wasi = new WASI(args, env, fds, { debug: false });
                const wasiImport = wasi.wasiImport;
                wasiImport['poll_oneoff'] = poll_oneoff;
                wasiImport['sock_accept'] = sock_accept;
                const wasi2 = new WASI(args, env, fds, { debug: false });
                const wasiImport2 = wasi2.wasiImport;
                wasiImport2['poll_oneoff'] = poll_oneoff;
                wasiImport2['sock_accept'] = sock_accept;
                // -------------------------------------------------------------------------------
                // -------------------------------------------------------------------------------
                let WASM_INSTANCE = null;
                const load_string = (ptr, len) => {
                    if (!WASM_INSTANCE) throw new Error('load_string: the wasm instance is missing');
                    const memory = new Uint8Array(WASM_INSTANCE.exports.memory.buffer);
                    const buf = memory.slice(ptr, ptr + len);
                    const decoder = new TextDecoder('utf-8');
                    const s = decoder.decode(buf);
                    return { buf, s };
                };
                const store_bytes = (bytes, ptr) => {
                    if (!WASM_INSTANCE) throw new Error('store_bytes: the wasm instance is missing');
                    const memory = new Uint8Array(WASM_INSTANCE.exports.memory.buffer);
                    memory.set(bytes, ptr);
                };
                let NEW_MODULE_ID = 41;
                const MODULE_MAP = {};
                const load_wasm_module = (wasmModulePathPtr, wasmModulePathLength) => {
                    const { s: wasmModulePath } = load_string(wasmModulePathPtr, wasmModulePathLength);
                    const preOpenedFd = fds[3];
                    console.log('[DEBUG] load_wasm_module called with path:', wasmModulePath, 'preOpenedFd:', preOpenedFd);
                    let currDirectoryOrFile = preOpenedFd.dir.contents;
                    wasmModulePath.split('/').forEach(p => {
                        if (p === '') return;
                        // console.log('looking for folder/file', p);
                        if (!(p in currDirectoryOrFile)) {
                            console.error('load_wasm_module: p', p, 'currDirectoryOrFile', currDirectoryOrFile);
                            throw new Error('load_wasm_module: failed to find the wasm module');
                        }
                        // console.log('before currDirectoryOrFile', currDirectoryOrFile, typeof currDirectoryOrFile);
                        currDirectoryOrFile = currDirectoryOrFile[p];
                        if (currDirectoryOrFile instanceof PreopenDirectory) {
                            currDirectoryOrFile = currDirectoryOrFile.dir.contents;
                        } else if (currDirectoryOrFile instanceof Directory) {
                            currDirectoryOrFile = currDirectoryOrFile.contents;
                        }
                        // console.log('after currDirectoryOrFile', currDirectoryOrFile);
                    });
                    if (!(currDirectoryOrFile instanceof File)) throw new Error('load_wasm_module: the given path is not a file');
                    const wasmModuleBytes = currDirectoryOrFile.data;
                    console.log('load_wasm_module: wasmModuleBytes', wasmModuleBytes);
                    // https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/Module/Module
                    const myModule = new WebAssembly.Module(wasmModuleBytes);
                    // const importObject = {
                    //     'console': {
                    //         'log': console.log
                    //     }
                    // };
                    const importObject = {
                        // [WASI_PREVIEW_1]: wasiImport,
                        [WASI_PREVIEW_1]: wasiImport2,
                    };
                    // https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/Instance/Instance
                    const instance = new WebAssembly.Instance(myModule, importObject);
                    console.log('load_wasm_module wasi.start start');
                    // wasi.start(instance);
                    wasi2.start(instance);
                    console.log('load_wasm_module wasi.start done');
                    const new_key = ++NEW_MODULE_ID;
                    console.log('load_wasm_module: compiled wasm and made an instance:', instance, 'module id:', new_key);
                    MODULE_MAP[new_key] = instance;
                    return new_key;
                };
                const run_transform = (is_dir_detect) => (wasmModuleId, inputJsonPtr, inputJsonLength, outputJsonPtr) => {
                    if (!(wasmModuleId in MODULE_MAP)) throw new Error(`There is no wasm module with id ${wasmModuleId}`);
                    // debugger;
                    console.log('[DEBUG] run_transform is_dir_detect:', is_dir_detect);
                    const wasmModule = MODULE_MAP[wasmModuleId];
                    const { buf, s } = load_string(inputJsonPtr, inputJsonLength);
                    // console.log('run_transform: load_string buf', buf, 's', s);
                    console.log('run_transform: load_string buf', buf);
                    const input = JSON.parse(s); // DEBUG: just to make sure it is json parseable
                    // console.log('run_transform called with: wasmModuleId:', wasmModuleId, 'wasmModule', wasmModule, 's:', s, 'input:', input);
                    console.log('run_transform called with: wasmModuleId:', wasmModuleId, 'wasmModule', wasmModule);
                    console.log('wasmModule.exports.myAllocate', wasmModule.exports.myAllocate);
                    console.log('wasmModule.exports.RunDirectoryDetect', wasmModule.exports.RunDirectoryDetect);
                    console.log('wasmModule.exports.RunTransform', wasmModule.exports.RunTransform);
                    // const len = s.length;
                    const len = buf.byteLength;
                    console.log('run_transform: allocate some memory of size:', len);
                    const ptr = wasmModule.exports.myAllocate(len);
                    console.log('run_transform: ptr', ptr, 'len', len);
                    if (ptr < 0) throw new Error('failed to allocate, invalid pointer into memory');
                    let memory = new Uint8Array(wasmModule.exports.memory.buffer);
                    memory.set(buf, ptr);
                    console.log('run_transform: json input set at ptr', ptr);
                    console.log('run_transform: allocate space for the output pointers');
                    const ptrptr = wasmModule.exports.myAllocate(8); // 2 uint32 values
                    console.log('run_transform: ptrptr', ptrptr);
                    if (ptrptr < 0) throw new Error('failed to allocate, invalid pointer into memory');
                    if (is_dir_detect) {
                        console.log('calling custom transformer directory detect');
                        const result = wasmModule.exports.RunDirectoryDetect(ptr, len, ptrptr, ptrptr + 4);
                        console.log('run_transform: directory detect result', result);
                        if (result < 0) throw new Error('run_transform: directory detect failed');
                    } else {
                        console.log('calling custom transformer transform');
                        const result = wasmModule.exports.RunTransform(ptr, len, ptrptr, ptrptr + 4);
                        console.log('run_transform: transformation result', result);
                        if (result < 0) throw new Error('run_transform: transformation failed');
                    }
                    const outJsonPtr = new DataView(wasmModule.exports.memory.buffer, ptrptr, 4).getUint32(0, true);
                    const outJsonLen = new DataView(wasmModule.exports.memory.buffer, ptrptr + 4, 4).getUint32(0, true);
                    console.log('run_transform: transformation outJsonPtr', outJsonPtr, 'outJsonLen', outJsonLen);
                    memory = new Uint8Array(wasmModule.exports.memory.buffer);
                    console.log('run_transform: memory', memory);
                    const outJsonBytes = memory.slice(outJsonPtr, outJsonPtr + outJsonLen);
                    console.log('run_transform: outJsonBytes', outJsonBytes);
                    const outJson = new TextDecoder('utf-8').decode(outJsonBytes);
                    console.log('run_transform: outJson', outJson);
                    const outJsonParsed = JSON.parse(outJson);
                    console.log('run_transform: outJsonParsed', outJsonParsed);
                    store_bytes(outJsonBytes, outputJsonPtr);
                    return outJsonBytes.length;
                };
                const importObject = {
                    [WASI_PREVIEW_1]: wasiImport,
                    "mym2kmodule": {
                        "load_wasm_module": load_wasm_module,
                        "run_dir_detect": run_transform(true),
                        "run_transform": run_transform(false),
                    },
                };
                // console.log('worker: importObject.wasi_snapshot_preview1', importObject.wasi_snapshot_preview1);
                // -------------------------------------------------------------------------------
                // -------------------------------------------------------------------------------
                const wasmModuleInstance = await WebAssembly.instantiate(wasmModule, importObject);
                WASM_INSTANCE = wasmModuleInstance;
                console.log('worker: wasmModuleInstance', wasmModuleInstance);
                console.log('worker: wasmModuleInstance.exports', wasmModuleInstance.exports);
                console.log('worker: wasmModuleInstance.exports.memory.buffer', wasmModuleInstance.exports.memory.buffer);
                try {
                    // wasi.start(wasmModule.instance);
                    wasi.start(wasmModuleInstance);
                    // TODO: unreachable?
                    self.postMessage({ 'type': MSG_TRANFORM_DONE, 'payload': 'transformation result (no exit code)' });
                } catch (e) {
                    console.log('worker: the wasm module finished:', e);
                    // console.log(typeof e);
                    // console.log(e.exit_code);
                    // console.log(Object.entries(e));
                    const eStr = `${e}`;
                    const exitCodePrefix = 'exit with exit code ';
                    if (eStr.startsWith(exitCodePrefix)) {
                        console.log('error message has exit code prefix');
                        const exitCodeStr = eStr.slice(exitCodePrefix.length);
                        // console.log('WOW!!!! exitCodeStr', exitCodeStr);
                        const exitCode = parseInt(exitCodeStr, 10);
                        console.log('exitCode', exitCode);
                        if (!Number.isFinite(exitCode) || exitCode !== 0) {
                            self.postMessage({ 'type': MSG_TRANFORM_ERR, 'payload': eStr });
                            console.log('ERROR non-zero exit code, DEBUG fds:', MY_DEBUG_FDS);
                            break;
                        }
                    } else {
                        console.log('error message does not have exit code prefix');
                    }
                    console.log('TODO: assuming the output file name is myproject.zip');
                    const myprojectzip = fds[3]?.dir?.contents["myproject.zip"]?.data?.buffer;
                    if (!myprojectzip) {
                        self.postMessage({ 'type': MSG_TRANFORM_ERR, 'payload': eStr });
                        console.log('ERROR myproject.zip is missing, DEBUG fds:', MY_DEBUG_FDS);
                        break;
                    }
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
