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
            // console.log('XtermStdio.fd_write msg:', msg);
            self.postMessage({ 'type': MSG_TERM_PRINT, 'payload': buffer });
            nwritten += iovec.buf_len;
        }
        return { ret: 0, nwritten };
    }
}

// CONSTANTS ---------------------------------------------------------------

const MSG_WASM_MODULE = 'wasm-module';
const MSG_TERM_PRINT = 'terminal-print';
const MSG_ASK_QUESTION = 'ask-question';
const MSG_TRANFORM_DONE = 'transform-done';
const MSG_TRANFORM_ERR = 'transform-error';
const MSG_INIT_WORKER = 'initialize-worker';
const MSG_WORKER_INITIALIZED = 'worker-initialized';
const WASI_PREVIEW_1 = 'wasi_snapshot_preview1';
const OUTPUT_DIR = 'my-m2k-output';

let SHARED_BUF_UINT8 = null;
let SHARED_BUF_INT32 = null;
let MY_DEBUG_FDS = null;

const METRICS_FD_HOST_FUNC_NAMES = [
    'fd_advise', 'fd_allocate', 'fd_close', 'fd_datasync', 'fd_fdstat_get', 'fd_fdstat_set_flags',
    'fd_fdstat_set_rights', 'fd_filestat_get', 'fd_filestat_set_size', 'fd_filestat_set_times', 'fd_pread', 'fd_prestat_dir_name',
    'fd_prestat_get', 'fd_pwrite', 'fd_read', 'fd_readdir', 'fd_renumber', 'fd_seek', 'fd_sync', 'fd_tell', 'fd_write',
];
const METRICS_IO_HOST_FUNC_NAMES = ['fd_readdir', 'fd_read', 'fd_pread', 'fd_write', 'fd_pwrite'];
const METRICS = {
    load_time: 0,
    execution_time: 0,
    memory_usage_start: 0,
    memory_usage_end: 0,
    io_count: 0,
    io_time: 0,
    call_counts: {},
    call_durations: {},
    custom_transformer_times: {},
};

// FUNCTIONS ---------------------------------------------------------------

const newCustMetricObj = (id, path) => ({
    id,
    path,
    countAllocate: 0,
    countDirDetect: 0,
    countTransform: 0,
    tCompile: 0,
    tInstantiate: 0,
    tStart: 0,
    tAllocate: 0,
    tDirDetect: 0,
    tTransform: 0,
    tLoad: 0,
    tExec: 0,
});

const resetMetrics = () => {
    METRICS.load_time = 0;
    METRICS.execution_time = 0;
    METRICS.memory_usage_start = 0;
    METRICS.memory_usage_end = 0;
    METRICS.io_count = 0;
    METRICS.io_time = 0;
    METRICS.call_counts = {};
    METRICS.call_durations = {};
    METRICS_FD_HOST_FUNC_NAMES.forEach(name => {
        METRICS.call_counts[name] = 0;
        METRICS.call_durations[name] = 0;
    });
    const newMet = {};
    for (const id in METRICS.custom_transformer_times) {
        const obj = METRICS.custom_transformer_times[id];
        newMet[id] = newCustMetricObj(id, obj.path);
        newMet[id].tCompile = obj.tCompile;
        newMet[id].tInstantiate = obj.tInstantiate;
        newMet[id].tStart = obj.tStart;
    }
    METRICS.custom_transformer_times = newMet;
};

resetMetrics();

const proxyHostFn = (impObj) => {
    console.log('proxyHostFn start self.crossOriginIsolated', self.crossOriginIsolated);
    // console.log('DEBUG impObj', impObj);
    METRICS_FD_HOST_FUNC_NAMES.forEach(name => {
        const host_fn = impObj[name];
        impObj[name] = (...args) => {
            // console.log(name);
            METRICS.call_counts[name]++;
            // https://stackoverflow.com/questions/29700256/is-performance-now-in-web-workers-reliable
            const tStart = performance.now();
            const results = host_fn(...args);
            const tEnd = performance.now();
            const tDuration = tEnd - tStart;
            // const tDurationSeconds = tDuration / 1000;
            // console.log('name', name, 'tStart', tStart, 'tEnd', tEnd, 'tDuration', tDuration, 'tDurationSeconds', tDurationSeconds);
            METRICS.call_durations[name] += tDuration;
            return results;
        };
    });
    console.log('proxyHostFn end');
};

const printMetrics = () => {
    console.log('printMetrics start');
    console.log('------------------------------------------------------------------');
    let total_count = 0;
    let total_duration = 0;
    METRICS_IO_HOST_FUNC_NAMES.forEach(name => {
        total_count += METRICS.call_counts[name];
        total_duration += METRICS.call_durations[name];
    });
    METRICS.io_count = total_count;
    METRICS.io_time = total_duration;
    console.log('calculate per transformer total times');
    for (const id in METRICS.custom_transformer_times) {
        const obj = METRICS.custom_transformer_times[id];
        obj.tLoad = obj.tCompile + obj.tInstantiate + obj.tStart;
        obj.tExec = obj.tAllocate + obj.tDirDetect + obj.tTransform;
    }
    console.log('METRICS', JSON.stringify(METRICS, null, 4));
    console.log('------------------------------------------------------------------');
    console.log('printMetrics end');
};

const fromMsg = (uint8, int32) => {
    const len = int32[1];
    if (len <= 0) throw new Error('object length is zero or negative');
    const x = uint8.slice(8, 8 + len);
    const dec = new TextDecoder();
    const s = dec.decode(x);
    return JSON.parse(s);
};

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
    console.log('processMessage start');
    try {
        const msg = e.data;
        console.log('got a message:', msg);
        switch (msg.type) {
            case MSG_INIT_WORKER: {
                console.log('MSG_INIT_WORKER payload:', msg.payload);
                const { sab } = msg.payload;
                console.log('sab', sab);
                if (sab) {
                    SHARED_BUF_UINT8 = new Uint8Array(sab);
                    SHARED_BUF_INT32 = new Int32Array(sab);
                }
                self.postMessage({ 'type': MSG_WORKER_INITIALIZED });
                break;
            }
            case MSG_WASM_MODULE: {
                console.log('got a wasm module:', typeof msg.payload, msg.payload);
                const {
                    wasmModule, srcFilename, srcContents, custFilename,
                    custContents, configFilename, configContents, qaSkip, enableMetrics,
                } = msg.payload;
                const args = ["move2kube", "transform", "--source", srcFilename, "--output", OUTPUT_DIR];
                const preOpenDir = {
                    [srcFilename]: new File(srcContents),
                };
                if (qaSkip) {
                    args.push("--qa-skip");
                }
                if (custFilename && custContents) {
                    args.push("--customizations", custFilename);
                    preOpenDir[custFilename] = new File(custContents);
                }
                if (configFilename && configContents) {
                    args.push("--config", configFilename);
                    preOpenDir[configFilename] = new File(configContents);
                }
                const env = [];
                const fds = [
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
                if (enableMetrics) proxyHostFn(wasiImport);

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
                let NEW_MODULE_ID = 0;
                const MODULE_MAP = {};
                const load_wasm_module = (wasmModulePathPtr, wasmModulePathLength) => {
                    const new_wasm_module_key = NEW_MODULE_ID + 1; // don't upadte the counter until compilation/instantiation succeeds
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
                    const tCustCompileStart = performance.now();
                    const myModule = new WebAssembly.Module(wasmModuleBytes);
                    const tCustCompileEnd = performance.now();
                    const tCustCompile = tCustCompileEnd - tCustCompileStart;
                    METRICS.custom_transformer_times[new_wasm_module_key] = newCustMetricObj(new_wasm_module_key, wasmModulePath);
                    METRICS.custom_transformer_times[new_wasm_module_key].tCompile = tCustCompile;
                    // const importObject = {
                    //     'console': {
                    //         'log': console.log
                    //     }
                    // };

                    // --------------------------------------------------
                    console.log('create a personal WASI instance for the custom wasm module/transformer');
                    const wasi2 = new WASI(args, env, fds, { debug: false });
                    const wasiImport2 = wasi2.wasiImport;
                    wasiImport2['poll_oneoff'] = poll_oneoff;
                    wasiImport2['sock_accept'] = sock_accept;
                    if (enableMetrics) proxyHostFn(wasiImport2);
                    // --------------------------------------------------

                    const importObject = {
                        // [WASI_PREVIEW_1]: wasiImport,
                        [WASI_PREVIEW_1]: wasiImport2,
                    };
                    // https://developer.mozilla.org/en-US/docs/WebAssembly/JavaScript_interface/Instance/Instance
                    const tCustInstantiateStart = performance.now();
                    const instance = new WebAssembly.Instance(myModule, importObject);
                    const tCustInstantiateEnd = performance.now();
                    const tCustInstantiate = tCustInstantiateEnd - tCustInstantiateStart;
                    METRICS.custom_transformer_times[new_wasm_module_key].tInstantiate = tCustInstantiate;
                    console.log('load_wasm_module wasi.start start');
                    // wasi.start(instance);
                    const tCustStart = performance.now();
                    wasi2.start(instance);
                    const tCustEnd = performance.now();
                    const tCust = tCustEnd - tCustStart;
                    METRICS.custom_transformer_times[new_wasm_module_key].tStart = tCust;
                    console.log('load_wasm_module wasi.start done');
                    console.log('load_wasm_module: compiled wasm and made an instance:', instance, 'module id:', new_wasm_module_key);
                    NEW_MODULE_ID = new_wasm_module_key; // compilation/instantiation succeeded so update the counter
                    MODULE_MAP[new_wasm_module_key] = instance;
                    return new_wasm_module_key;
                };
                const run_transform = (is_dir_detect) => (wasmModuleId, inputJsonPtr, inputJsonLength, outputJsonPtr) => {
                    console.log('run_transform start is_dir_detect:', is_dir_detect);
                    if (!(wasmModuleId in MODULE_MAP)) throw new Error(`There is no wasm module with id ${wasmModuleId}`);
                    const customTransformerWasmModule = MODULE_MAP[wasmModuleId];
                    const { buf, s } = load_string(inputJsonPtr, inputJsonLength);
                    // console.log('run_transform: load_string buf', buf, 's', s);
                    console.log('run_transform: load_string buf', buf);
                    const input = JSON.parse(s); // DEBUG: just to make sure it is json parseable
                    // console.log('run_transform called with: wasmModuleId:', wasmModuleId, 'wasmModule', wasmModule, 's:', s, 'input:', input);
                    console.log('run_transform called with: wasmModuleId:', wasmModuleId, 'wasmModule', customTransformerWasmModule);
                    // console.log('wasmModule.exports.myAllocate', customTransformerWasmModule.exports.myAllocate);
                    // console.log('wasmModule.exports.RunDirectoryDetect', customTransformerWasmModule.exports.RunDirectoryDetect);
                    // console.log('wasmModule.exports.RunTransform', customTransformerWasmModule.exports.RunTransform);
                    // const len = s.length;
                    const len = buf.byteLength;
                    console.log('run_transform: allocate some memory of size:', len);

                    const tCustAllocateStart = performance.now();
                    const ptr = customTransformerWasmModule.exports.myAllocate(len);
                    const tCustAllocateEnd = performance.now();
                    const tCustAllocate = tCustAllocateEnd - tCustAllocateStart;
                    METRICS.custom_transformer_times[wasmModuleId].tAllocate += tCustAllocate;
                    METRICS.custom_transformer_times[wasmModuleId].countAllocate++;

                    console.log('run_transform: ptr', ptr, 'len', len);
                    if (ptr < 0) throw new Error('failed to allocate, invalid pointer into memory');
                    let memory = new Uint8Array(customTransformerWasmModule.exports.memory.buffer);
                    memory.set(buf, ptr);
                    console.log('run_transform: json input set at ptr', ptr);
                    console.log('run_transform: allocate space for the output pointers');
                    const ptrptr = customTransformerWasmModule.exports.myAllocate(8); // 2 uint32 values
                    console.log('run_transform: ptrptr', ptrptr);
                    if (ptrptr < 0) throw new Error('failed to allocate, invalid pointer into memory');
                    if (is_dir_detect) {
                        console.log('calling custom transformer directory detect');

                        const tCustDetectStart = performance.now();
                        const result = customTransformerWasmModule.exports.RunDirectoryDetect(ptr, len, ptrptr, ptrptr + 4);
                        const tCustDetectEnd = performance.now();
                        const tCustDetect = tCustDetectEnd - tCustDetectStart;
                        METRICS.custom_transformer_times[wasmModuleId].tDirDetect += tCustDetect;
                        METRICS.custom_transformer_times[wasmModuleId].countDirDetect++;

                        console.log('run_transform: directory detect result', result);
                        if (result < 0) throw new Error('run_transform: directory detect failed');
                    } else {
                        console.log('calling custom transformer transform');

                        const tCustTranformStart = performance.now();
                        const result = customTransformerWasmModule.exports.RunTransform(ptr, len, ptrptr, ptrptr + 4);
                        const tCustTranformEnd = performance.now();
                        const tCustTranform = tCustTranformEnd - tCustTranformStart;
                        METRICS.custom_transformer_times[wasmModuleId].tTransform += tCustTranform;
                        METRICS.custom_transformer_times[wasmModuleId].countTransform++;

                        console.log('run_transform: transformation result', result);
                        if (result < 0) throw new Error('run_transform: transformation failed');
                    }
                    const outJsonPtr = new DataView(customTransformerWasmModule.exports.memory.buffer, ptrptr, 4).getUint32(0, true);
                    const outJsonLen = new DataView(customTransformerWasmModule.exports.memory.buffer, ptrptr + 4, 4).getUint32(0, true);
                    console.log('run_transform: transformation outJsonPtr', outJsonPtr, 'outJsonLen', outJsonLen);
                    memory = new Uint8Array(customTransformerWasmModule.exports.memory.buffer);
                    console.log('run_transform: memory', memory);
                    const outJsonBytes = memory.slice(outJsonPtr, outJsonPtr + outJsonLen);
                    console.log('run_transform: outJsonBytes', outJsonBytes);
                    const outJson = new TextDecoder('utf-8').decode(outJsonBytes);
                    console.log('run_transform: outJson', outJson);
                    const outJsonParsed = JSON.parse(outJson);
                    console.log('run_transform: outJsonParsed', outJsonParsed);
                    store_bytes(outJsonBytes, outputJsonPtr);
                    console.log('run_transform end');
                    return outJsonBytes.length;
                };
                const ask_question = (inputJsonPtr, inputJsonLength, outputJsonPtr) => {
                    if (outputJsonPtr < 0) throw new Error('the output pointer is an invalid pointer into memory');
                    const { s } = load_string(inputJsonPtr, inputJsonLength);
                    // console.log('ask_question: load_string buf', buf, 's', s);
                    const ques = JSON.parse(s);
                    console.log('ask the main thread to ask the question:', ques);
                    self.postMessage({
                        'type': MSG_ASK_QUESTION,
                        'payload': ques,
                    });
                    console.log('ask_question: wait until the question is answered');
                    const waitOk = Atomics.wait(SHARED_BUF_INT32, 0, 0);
                    console.log('ask_question: waitOk:', waitOk);
                    if (waitOk !== 'ok') throw new Error(`Atomics.wait failed waitOk: ${waitOk}`);
                    // const len = s.length;
                    const ans = fromMsg(SHARED_BUF_UINT8, SHARED_BUF_INT32);
                    console.log('got an answer from main thread, ans:', ans);
                    const ansStr = JSON.stringify(ans);
                    // console.log('ansStr:', ansStr);
                    const enc = new TextEncoder();
                    const ansBytes = enc.encode(ansStr);
                    // console.log('ansBytes:', ansBytes);
                    store_bytes(ansBytes, outputJsonPtr);
                    return ansBytes.byteLength;
                };
                const importObject = {
                    [WASI_PREVIEW_1]: wasiImport,
                    "mym2kmodule": {
                        "load_wasm_module": load_wasm_module,
                        "run_dir_detect": run_transform(true),
                        "run_transform": run_transform(false),
                        "ask_question": ask_question,
                    },
                };
                // console.log('importObject.wasi_snapshot_preview1', importObject.wasi_snapshot_preview1);
                // -------------------------------------------------------------------------------
                // -------------------------------------------------------------------------------
                resetMetrics();
                const tInstantiateStart = performance.now();
                const wasmModuleInstance = await WebAssembly.instantiate(wasmModule, importObject);
                const tInstantiateEnd = performance.now();
                const tInstantiate = tInstantiateEnd - tInstantiateStart;
                const tInstantiateSeconds = tInstantiate / 1000;
                METRICS.load_time = tInstantiate;
                console.log('tInstantiateStart', tInstantiateStart, 'tInstantiateEnd', tInstantiateEnd, 'tInstantiate', tInstantiate, 'tInstantiateSeconds', tInstantiateSeconds);
                WASM_INSTANCE = wasmModuleInstance;
                console.log('wasmModuleInstance', wasmModuleInstance);
                console.log('wasmModuleInstance.exports', wasmModuleInstance.exports);
                console.log('wasmModuleInstance.exports.memory.buffer', wasmModuleInstance.exports.memory.buffer);
                console.log('wasmModuleInstance.exports.memory.buffer.byteLength', wasmModuleInstance.exports.memory.buffer.byteLength);
                METRICS.memory_usage_start = wasmModuleInstance.exports.memory.buffer.byteLength;
                try {
                    // wasi.start(wasmModule.instance);
                    const tTransformStart = performance.now();
                    const exitCode = wasi.start(wasmModuleInstance);
                    const tTransformEnd = performance.now();
                    const tTransform = tTransformEnd - tTransformStart;
                    const tTransformSeconds = tTransform / 1000;
                    METRICS.execution_time = tTransform;
                    console.log('tTransformStart', tTransformStart, 'tTransformEnd', tTransformEnd, 'tTransform', tTransform, 'tTransformSeconds', tTransformSeconds);
                    console.log('exitCode:', exitCode);
                    if (exitCode !== 0) {
                        const eStr = `got a non-zero exit code: ${exitCode}`;
                        console.error(eStr, 'DEBUG fds:', MY_DEBUG_FDS);
                        self.postMessage({ 'type': MSG_TRANFORM_ERR, 'payload': eStr });
                        break;
                    }
                    const myprojectzip = fds[3]?.dir?.contents["myproject.zip"]?.data?.buffer;
                    if (!myprojectzip) {
                        self.postMessage({ 'type': MSG_TRANFORM_ERR, 'payload': 'The output "myproject.zip" file is missing.' });
                        console.log('ERROR myproject.zip is missing, DEBUG fds:', MY_DEBUG_FDS);
                        break;
                    }
                    self.postMessage({
                        'type': MSG_TRANFORM_DONE,
                        'payload': { myprojectzip, tTransform },
                    });
                    console.log('after transformation wasmModuleInstance.exports.memory.buffer.byteLength', wasmModuleInstance.exports.memory.buffer.byteLength);
                    METRICS.memory_usage_end = wasmModuleInstance.exports.memory.buffer.byteLength;
                    printMetrics();
                } catch (e) {
                    console.error('the wasm module finished with an error:', e, 'DEBUG fds:', MY_DEBUG_FDS);
                    const eStr = `${e}`;
                    self.postMessage({ 'type': MSG_TRANFORM_ERR, 'payload': eStr });
                }
                break;
            }
            default: {
                throw new Error(`unknown message type: ${msg.type}`);
            }
        }
    } catch (e) {
        console.error('failed to process the message. error:', e);
    }
    console.log('processMessage end');
};

const main = () => {
    const prevConsoleLog = console.log;
    console.log = (...args) => prevConsoleLog('[worker]', ...args);
    const prevConsoleErr = console.error;
    console.error = (...args) => prevConsoleErr('[worker]', ...args);
    console.log('main start');
    self.addEventListener('message', processMessage);
    console.log('main end');
};

main();
