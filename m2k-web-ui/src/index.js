import 'xterm/css/xterm.css';

import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import pako from 'pako';
import axios from 'axios';

// CONSTANTS ---------------------------------------------------------------

const SHARED_BUF_LENGTH = 65536;

const QUES_TYPE_INPUT = 'Input';
const QUES_TYPE_INPUT_MULTI_LINE = 'MultiLineInput';
const QUES_TYPE_MULTI = 'MultiSelect';
const QUES_TYPE_SELECT = 'Select';
const QUES_TYPE_CONFIRM = 'Confirm';
const QUES_TYPE_PASSWORD = 'Password';

const MSG_WASM_MODULE = 'wasm-module';
const MSG_TERM_PRINT = 'terminal-print';
const MSG_ASK_QUESTION = 'ask-question';
const MSG_TRANFORM_DONE = 'transform-done';
const MSG_TRANFORM_ERR = 'transform-error';
const MSG_INIT_WORKER = 'initialize-worker';
const MSG_WORKER_INITIALIZED = 'worker-initialized';

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

let SHARED_BUF_UINT8 = null;
let SHARED_BUF_INT32 = null;

// FUNCTIONS ---------------------------------------------------------------

const toMsg = (uint8, int32, obj) => {
    const s = JSON.stringify(obj);
    const enc = new TextEncoder();
    const x = enc.encode(s);
    if (x.byteLength + 8 > uint8.byteLength) {
        throw new Error('object length is larger than buffer length');
    }
    int32[1] = x.byteLength;
    uint8.set(x, 8);
};

function downloadArrayBufferAsBlob(arrayBuffer) {
    const bs = new Blob([arrayBuffer]);
    const ys = URL.createObjectURL(bs);
    const aelem = document.createElement('a');
    aelem.setAttribute('href', ys);
    aelem.download = 'myproject.zip';
    document.body.appendChild(aelem);
    aelem.click();
}

const handleSendAnswer = (ques, answer) => {
    const answerObj = {
        ...ques,
        'answer': answer,
    };
    toMsg(SHARED_BUF_UINT8, SHARED_BUF_INT32, answerObj);
    console.log('notify the worker about the answer to the question');
    const awoke = Atomics.notify(SHARED_BUF_INT32, 0);
    console.log('awoke', awoke);
};

const handleAskQuestion = (ques) => {
    console.log('handleAskQuestion ques:', ques);
    const div_modal = document.querySelector('#div-modal');
    const div_modal_ques_id = document.querySelector('#div-modal-ques-id');
    const div_modal_ques_desc = document.querySelector('#div-modal-ques-desc');
    const div_modal_body = document.querySelector('#div-modal-ques-body');
    const button_next = document.querySelector('#button-modal-confirm');
    div_modal_ques_id.textContent = `ID: ${ques.id}`;
    div_modal_ques_desc.textContent = ques.description;
    div_modal_body.innerHTML = '';
    // div_modal_ques_id.classList.add('border-bottom');
    // div_modal_body.appendChild(div_modal_ques_id);
    switch (ques.type) {
        case QUES_TYPE_PASSWORD:
        case QUES_TYPE_INPUT_MULTI_LINE:
        case QUES_TYPE_INPUT: {
            const inputText = document.createElement('input');
            inputText.setAttribute('type', 'text');
            inputText.classList.add('modal-text-input');
            if ('default' in ques) inputText.setAttribute('value', ques.default);
            div_modal_body.appendChild(inputText);

            button_next.addEventListener('click', () => {
                console.log('clicked confirm on input', inputText, inputText.value);
                const answer = inputText.value;
                handleSendAnswer(ques, answer);
                div_modal.classList.add('hidden');
            }, { once: true });
            div_modal.classList.remove('hidden');
            break;
        }
        case QUES_TYPE_MULTI: {
            const listOfOptions = document.createElement('ul');
            listOfOptions.classList.add('modal-multi-select');
            const elemChecks = [];
            for (let optionValue of ques.options) {
                const li = document.createElement('li');
                const label = document.createElement('label');
                label.textContent = optionValue;
                const check = document.createElement('input');
                check.setAttribute('type', 'checkbox');
                check.setAttribute('data-value', optionValue);
                if (ques.default?.includes(optionValue)) {
                    check.setAttribute('checked', 'true');
                }
                elemChecks.push(check);
                label.appendChild(check);
                li.appendChild(label);
                listOfOptions.appendChild(li);
            }
            div_modal_body.appendChild(listOfOptions);

            button_next.addEventListener('click', () => {
                const ansArr = [];
                for (let e of elemChecks) {
                    if (e.checked) {
                        ansArr.push(e.getAttribute('data-value'));
                    }
                }
                const answer = ansArr;
                handleSendAnswer(ques, answer);
                div_modal.classList.add('hidden');
            }, { once: true });
            div_modal.classList.remove('hidden');
            break;
        }
        case QUES_TYPE_SELECT: {
            const select = document.createElement('select');
            select.classList.add('modal-select');
            for (let optionValue of ques.options) {
                const selectOption = document.createElement('option');
                selectOption.setAttribute('value', optionValue);
                selectOption.textContent = optionValue;
                select.appendChild(selectOption);
                if (('default' in ques) && optionValue === ques.default) selectOption.setAttribute('selected', 'true');
            }
            div_modal_body.appendChild(select);

            button_next.addEventListener('click', () => {
                console.log('clicked confirm on select', select, select.value);
                const answer = select.value;
                handleSendAnswer(ques, answer);
                div_modal.classList.add('hidden');
            }, { once: true });
            div_modal.classList.remove('hidden');
            break;
        }
        case QUES_TYPE_CONFIRM: {
            const label = document.createElement('label');
            label.textContent = 'Agree';
            label.classList.add('modal-confirm');
            const check = document.createElement('input');
            check.setAttribute('type', 'checkbox');
            label.appendChild(check);
            if (('default' in ques) && ques.default) check.setAttribute('checked', 'true');
            div_modal_body.appendChild(label);

            button_next.addEventListener('click', () => {
                console.log('clicked confirm on confirm', check, check.checked);
                const answer = check.checked;
                handleSendAnswer(ques, answer);
                div_modal.classList.add('hidden');
            }, { once: true });
            div_modal.classList.remove('hidden');
            break;
        }
        default: {
            throw new Error('unknown question type');
        }
    }
};

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
            button_start.disabled = false; // unlock after doing the transformation
            transformation_status.textContent = '❌ Failed - ' + msg.payload;
            transformation_status.classList.remove(CLASS_SUCCESS);
            transformation_status.classList.remove(CLASS_RUNNING);
            transformation_status.classList.add(CLASS_FAILURE);
            transformation_status.classList.remove(CLASS_HIDDEN);
            break;
        }
        case MSG_ASK_QUESTION: {
            console.log('main: ask question');
            const ques = msg.payload;
            handleAskQuestion(ques);
            break;
        }
        default: {
            console.error('main: unknown worker message type:', msg);
        }
    }
};

const startWasmTransformation = async (
    srcFilename, srcContents, custFilename, custContents,
    configFilename, configContents, qaSkip,
) => {
    console.log('startWasmTransformation: send the WASM module and zip file to the web worker');
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
            'qaSkip': qaSkip,
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
    const input_qa_skip = document.getElementById('input-qa-skip');
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
        const qaSkip = input_qa_skip.checked;
        console.log('qaSkip', qaSkip);
        startWasmTransformation(srcFilename, contentsArr, custFilename, contentsCustArr, configFilename, contentsConfigArr, qaSkip);
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
    WASM_WEB_WORKER = wasmWorker;

    const workerInitPromise = new Promise((resolve) => {
        wasmWorker.addEventListener('message', (e) => {
            if (e.data.type !== MSG_WORKER_INITIALIZED) {
                throw new Error(`expected worker initialized message. actual: ${e.data.type}`);
            }
            resolve();
        }, { once: true });
    });
    if (window.SharedArrayBuffer) {
        console.log('SharedArrayBuffer is available, enabling the interactive QA engine');
        const sab = new SharedArrayBuffer(SHARED_BUF_LENGTH);
        console.log('sab', sab, 'window.crossOriginIsolated.', window.crossOriginIsolated);
        SHARED_BUF_UINT8 = new Uint8Array(sab);
        SHARED_BUF_INT32 = new Int32Array(sab);
        wasmWorker.postMessage({ 'type': MSG_INIT_WORKER, 'payload': { sab } });
    } else {
        console.log('SharedArrayBuffer is not available, disabling the interactive QA engine');
        const input_qa_skip = document.getElementById('input-qa-skip');
        input_qa_skip.setAttribute('checked', 'true');
        input_qa_skip.disabled = true;
        const label_input_qa_skip = document.getElementById('label-input-qa-skip');
        label_input_qa_skip.classList.add('hidden');
        wasmWorker.postMessage({ 'type': MSG_INIT_WORKER, 'payload': {} });
    }
    await workerInitPromise;
    console.log('worker initialized');
    wasmWorker.addEventListener('message', processWorkerMessage);

    console.log('fetching the Move2Kube WASM module');
    const progress = document.getElementById("fetch-progress");
    const progress_span = document.getElementById("fetch-progress-span");
    const axiosget = await axios.get(WASM_MODULE_URL, {
        responseType: 'arraybuffer',
        onDownloadProgress: (axiosProgressEvent) => {
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
