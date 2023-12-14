import 'xterm/css/xterm.css';

import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import pako from 'pako';
import axios from 'axios';

// CONSTANTS ---------------------------------------------------------------

const MSG_WASM_MODULE = 'wasm-module';
const MSG_TERM_PRINT = 'terminal-print';
const MSG_TRANFORM_DONE = 'transform-done';

const WASM_MODULE_URL = 'move2kube.wasm.gz';

// VARIABLES ---------------------------------------------------------------

let WASM_MODULE_COMPILED = null;
let WASM_WEB_WORKER = null;
let TERMINAL = null;
let TRANSFORM_RESULT = null;

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
            const btn_download = document.getElementById("button-download");
            btn_download.disabled = false;
            break;
        }
        default: {
            console.error('main: unknown worker message type:', msg);
        }
    }
};

const startWasmTransformation = async (filename, fileContentsArr) => {
    console.log('main: send the WASM module and zip file to the web worker');
    WASM_WEB_WORKER.postMessage({
        type: MSG_WASM_MODULE,
        payload: {
            'wasmModule': WASM_MODULE_COMPILED,
            'filename': filename,
            'fileContentsArr': fileContentsArr,
        },
    });
};

const addEventListeners = () => {
    const input_file = document.getElementById('input-file');
    input_file.addEventListener('change', async () => {
        if (!input_file.files || input_file.files.length === 0) return;
        console.log('got these files', input_file.files.length, input_file.files);
        const files = Array.from(input_file.files)
        if (files.length > 1) return console.error('only single file processing is supported for now');
        const f = files[0];
        console.log('reading the file named', f.name);
        const reader = new FileReader();
        const get_contents = new Promise((resolve, reject) => {
            reader.addEventListener('load', () => resolve(reader.result));
            reader.addEventListener('error', (e) => reject(e));
        });
        try {
            reader.readAsArrayBuffer(f);
            const contents = await get_contents;
            // console.log('input file contents:', contents);
            const contentsArr = new Uint8Array(contents);
            startWasmTransformation(f.name, contentsArr);
        } catch (e) {
            console.error(`failed to read the file '${f.name}' . error:`, e);
        }
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
    progress_label.classList.add("hidden");
    const input_file = document.getElementById("input-file");
    input_file.disabled = false;

    console.log('main end');
};

main().catch(console.error);
