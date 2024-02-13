import 'xterm/css/xterm.css';

import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import pako from 'pako';
import axios from 'axios';

// CONSTANTS ---------------------------------------------------------------

const MSG_WASM_MODULE = 'wasm-module';
const MSG_TERM_PRINT = 'terminal-print';
const MSG_TRANFORM_DONE = 'transform-done';
const MSG_TRANFORM_ERR = 'transform-error';

const WASM_MODULE_URL = 'move2kube.wasm.gz';
const CLASS_HIDDEN = 'hidden';
const CLASS_SUCCESS = 'success';
const CLASS_FAILURE = 'failure';
const CLASS_RUNNING = 'running';

// VARIABLES ---------------------------------------------------------------

let WASM_MODULE_COMPILED = null;
let WASM_WEB_WORKER = null;
let TERMINAL = null;
let TRANSFORM_RESULT = null;
let TRANSFORM_ERROR = null;

// FUNCTIONS ---------------------------------------------------------------

function downloadArrayBufferAsBlob(arrayBuffer) {
    const bs = new Blob([arrayBuffer]);
    const ys = URL.createObjectURL(bs);
    const aelem = document.createElement('a');
    aelem.setAttribute('href', ys);
    aelem.download = 'myproject.zip';
    document.body.appendChild(aelem);
    aelem.click();
}

const processWorkerMessage = async (e) => {
    const msg = e.data;
    const button_start = document.getElementById('button-start-transformation');
    const transformation_status = document.getElementById('transformation-status');
    // console.log('main: got a message from worker:', msg);
    switch (msg.type) {
        case MSG_TERM_PRINT: {
            // console.log('main: print something to terminal');
            // console.log(msg.payload);
            TERMINAL.write(msg.payload);
            break;
        }
        case MSG_TRANFORM_DONE: {
            console.log('main: transformation finished');
            TRANSFORM_RESULT = msg.payload;
            TRANSFORM_ERROR = null;
            const btn_download = document.getElementById("button-download");
            btn_download.disabled = false;
            button_start.disabled = false; // unlock after doing the transformation
            transformation_status.textContent = '✅ Success';
            transformation_status.classList.remove(CLASS_FAILURE);
            transformation_status.classList.remove(CLASS_RUNNING);
            transformation_status.classList.add(CLASS_SUCCESS);
            transformation_status.classList.remove(CLASS_HIDDEN);
            break;
        }
        case MSG_TRANFORM_ERR: {
            console.log('main: transformation error');
            TRANSFORM_RESULT = null;
            TRANSFORM_ERROR = msg.payload;
            button_start.disabled = false; // unlock after doing the transformation
            transformation_status.textContent = '❌ Failed - ' + msg.payload;
            transformation_status.classList.remove(CLASS_SUCCESS);
            transformation_status.classList.remove(CLASS_RUNNING);
            transformation_status.classList.add(CLASS_FAILURE);
            transformation_status.classList.remove(CLASS_HIDDEN);
            break;
        }
        default: {
            console.error('main: unknown worker message type:', msg);
        }
    }
};

const startWasmTransformation = async (srcFilename, srcContents, custFilename, custContents, configFilename, configContents) => {
    console.log('main: send the WASM module and zip file to the web worker');
    WASM_WEB_WORKER.postMessage({
        type: MSG_WASM_MODULE,
        payload: {
            'wasmModule': WASM_MODULE_COMPILED,
            'srcFilename': srcFilename,
            'srcContents': srcContents,
            'custFilename': custFilename,
            'custContents': custContents,
            'configFilename': configFilename,
            'configContents': configContents,
        },
    });
};

const readFileAsync = (f) => {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.addEventListener('load', () => resolve(reader.result));
        reader.addEventListener('error', (e) => reject(e));
        reader.readAsArrayBuffer(f);
    });
};

const addEventListeners = () => {
    const input_file = document.getElementById('input-file');
    const input_file_cust = document.getElementById('input-file-customizations');
    const input_file_config = document.getElementById('input-file-config');
    const button_start = document.getElementById('button-start-transformation');
    let srcFilename, contentsArr, custFilename, contentsCustArr, configFilename, contentsConfigArr;
    input_file.addEventListener('change', async () => {
        if (!input_file.files || input_file.files.length === 0) return;
        console.log('got these files', input_file.files.length, input_file.files);
        const files = Array.from(input_file.files);
        if (files.length > 1) return console.error('only single file processing is supported for now');
        const f = files[0];
        console.log('reading the file named', f.name);
        try {
            button_start.disabled = true; // lock while reading the file
            // const contentsArr = new Uint8Array(await readFileAsync(f));
            contentsArr = new Uint8Array(await readFileAsync(f));
            srcFilename = f.name;
            // console.log('input source file contentsArr:', contentsArr);
            input_file_cust.disabled = false; // TODO: support customizations without a source archive
            input_file_config.disabled = false; // TODO: support config + customizations without a source archive
            button_start.disabled = false;
        } catch (e) {
            console.error(`failed to read the source archive file '${f.name}' . error:`, e);
        }
    });
    input_file_cust.addEventListener('change', async () => {
        if (!input_file_cust.files || input_file_cust.files.length === 0) return;
        console.log('got these files', input_file_cust.files.length, input_file_cust.files);
        const files = Array.from(input_file_cust.files);
        if (files.length > 1) return console.error('only single file processing is supported for now');
        const f = files[0];
        console.log('reading the file named', f.name);
        try {
            button_start.disabled = true; // lock while reading the file
            // const contentsArr = new Uint8Array(await readFileAsync(f));
            contentsCustArr = new Uint8Array(await readFileAsync(f));
            custFilename = f.name;
            button_start.disabled = false;
            // console.log('input customizations file contentsArr:', contentsArr);
        } catch (e) {
            console.error(`failed to read the customizations archive file '${f.name}' . error:`, e);
        }
    });
    input_file_config.addEventListener('change', async () => {
        if (!input_file_config.files || input_file_config.files.length === 0) return;
        console.log('got these files', input_file_config.files.length, input_file_config.files);
        const files = Array.from(input_file_config.files);
        if (files.length > 1) return console.error('only single file processing is supported for now');
        const f = files[0];
        console.log('reading the file named', f.name);
        try {
            button_start.disabled = true; // lock while reading the file
            // const contentsArr = new Uint8Array(await readFileAsync(f));
            contentsConfigArr = new Uint8Array(await readFileAsync(f));
            configFilename = f.name;
            button_start.disabled = false;
            // console.log('input customizations file contentsArr:', contentsArr);
        } catch (e) {
            console.error(`failed to read the config yaml file '${f.name}' . error:`, e);
        }
    });
    button_start.addEventListener('click', () => {
        button_start.disabled = true; // lock while doing the transformation
        const transformation_status = document.getElementById('transformation-status');
        transformation_status.classList.remove(CLASS_SUCCESS);
        transformation_status.classList.remove(CLASS_FAILURE);
        transformation_status.classList.add(CLASS_RUNNING);
        transformation_status.classList.remove(CLASS_HIDDEN);
        transformation_status.textContent = '⚡ Running...';
        startWasmTransformation(srcFilename, contentsArr, custFilename, contentsCustArr, configFilename, contentsConfigArr);
    });
    const btn_download = document.getElementById('button-download');
    btn_download.addEventListener("click", () => {
        if (!TRANSFORM_RESULT) throw new Error('no transformation result');
        downloadArrayBufferAsBlob(TRANSFORM_RESULT);
    });

    // create the terminal object and attach it to the element
    const rootE = document.getElementById("div-root");
    const term = new Terminal({ convertEol: true });
    // console.log('term', term);
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(rootE);
    fitAddon.fit();
    TERMINAL = term;
};

const main = async () => {
    console.log('main start');

    addEventListeners();

    // start a web worker that can handle the transformation requests
    if (!window.Worker) {
        const err = 'Web Workers are not supported';
        alert('Web Workers are not supported');
        throw new Error(err);
    }
    const wasmWorker = new Worker(new URL('./worker.js', import.meta.url));
    console.log('wasmWorker', wasmWorker);
    wasmWorker.addEventListener('message', processWorkerMessage);
    WASM_WEB_WORKER = wasmWorker;

    console.log('fetching the Move2Kube WASM module');
    const progress = document.getElementById("fetch-progress");
    const progress_span = document.getElementById("fetch-progress-span");
    const axiosget = await axios.get(WASM_MODULE_URL, {
        responseType: 'arraybuffer',
        onDownloadProgress: function (axiosProgressEvent) {
            // console.log(axiosProgressEvent);
            progress.value = Math.trunc(axiosProgressEvent.progress * 10000) / 100;
            progress_span.textContent = `${progress.value}%`;
        }
    });

    // expand the gzip compressed archive and compile the WASM module
    const moduleObject = pako.inflate(new Uint8Array(axiosget.data));
    console.log('typeof moduleObject', typeof moduleObject, moduleObject);
    const compiledWasmModule = await WebAssembly.compile(moduleObject);
    console.log('typeof compiledWasmModule', typeof compiledWasmModule, compiledWasmModule);
    WASM_MODULE_COMPILED = compiledWasmModule;

    // enable the UI controls so the user can upload the input
    const progress_label = document.querySelector(".fetch-progress-label");
    progress_label.classList.add(CLASS_HIDDEN);
    const input_file = document.getElementById("input-file");
    input_file.disabled = false;

    console.log('main end');
};

main().catch(console.error);
