import 'xterm/css/xterm.css';

import { WASI, Fd, File, OpenFile, PreopenDirectory } from "@bjorn3/browser_wasi_shim";
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';

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

var FILE_SYSTEM;

function downloadArrayBufferAsBlob(arrayBuffer) {
    const bs = new Blob([arrayBuffer]);
    const ys=URL.createObjectURL(bs);
    const aelem = document.createElement('a');
    aelem.setAttribute('href', ys);
    aelem.download = 'myproject.zip';
    document.body.appendChild(aelem);
    aelem.click();
}

const start_wasm = async (rootE, filename, fileContentsArr) => {
    // create terminal object and attach to the element
    const term = new Terminal({
        convertEol: true,
    });
    console.log('term', term);
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(rootE);
    fitAddon.fit();

    // terminal as a file descriptor
    const encoder = new TextEncoder();
    const decoder = new TextDecoder();
    class XtermStdio extends Fd {
        constructor(term/*: Terminal*/) {
            super();
            this.term = term;
        }
        fd_write(view8/*: Uint8Array*/, iovs/*: [wasi.Iovec]*/)/*: {ret: number, nwritten: number}*/ {
            let nwritten = 0;
            for (let iovec of iovs) {
                // console.log(iovec.buf, iovec.buf_len, view8.slice(iovec.buf, iovec.buf + iovec.buf_len));
                const buffer = view8.slice(iovec.buf, iovec.buf + iovec.buf_len);
                const msg = decoder.decode(buffer);
                console.log('XtermStdio.fd_write msg', msg);
                // this.term.writeUtf8(buffer);
                // this.term.write(msg);
                this.term.write(buffer);
                nwritten += iovec.buf_len;
            }
            return { ret: 0, nwritten };
        }
    }

    // const args = ["move2kube", "-h"];
    // const args = ["move2kube", "version", "-l"];
    // const args = ["move2kube", "plan"];
    const args = ["move2kube", "plan", "-s", filename];
    const env = [];
    // const env = ["FOO=bar", "MYPWD=/"];
    // const env = ["FOO=bar", "PWD=/", "MYPWD=/"];
    // const env = ["FOO=bar", "PWD=.", "MYPWD=."];
    // const env = ["FOO=bar", "PWD=app", "MYPWD=app"];
    const fds = [
        // new OpenFile(new File([])), // stdin
        // new OpenFile(new File([])), // stdout
        // new OpenFile(new File([])), // stderr
        new XtermStdio(term), // stdin
        new XtermStdio(term), // stdout
        new XtermStdio(term), // stderr
        new PreopenDirectory("/", {
            "example.c": new File(encoder.encode(`#include "a"`)),
            "hello.rs": new File(encoder.encode(`fn main() { println!("Hello World!"); }`)),
            "dep.json": new File(encoder.encode(`{"a": 42, "b": 12}`)),
            [filename]: new File(fileContentsArr),
        }),
    ];
    FILE_SYSTEM = fds
    const wasi = new WASI(args, env, fds);

    const importObject = {
        "wasi_snapshot_preview1": wasi.wasiImport,
    };
    importObject.wasi_snapshot_preview1['poll_oneoff'] = poll_oneoff;
    importObject.wasi_snapshot_preview1['sock_accept'] = sock_accept;
    console.log('importObject.wasi_snapshot_preview1', importObject.wasi_snapshot_preview1);
    const all_wasi_host_func_names = Object.keys(importObject.wasi_snapshot_preview1);
    console.log('all_wasi_host_func_names', all_wasi_host_func_names);
    all_wasi_host_func_names.forEach(k => {
        const orig = importObject.wasi_snapshot_preview1[k];
        importObject.wasi_snapshot_preview1[k] = (...args) => {
            // https://wasix.org/docs/api-reference/wasi/path_open
            // dirfd dirflags path path_len o_flags fs_rights_base fs_rights_inheriting fs_flags fd
            // proxy for path_open !! -1 1 21021328 8 0 267910846n 268435455n 0 21281856
            // proxy for path_open !! -1 1 21021328 8 0 267910846n 268435455n 0 21281856
            // proxy for path_open !! -1 1 21021328 8 0 267910846n 268435455n 0 21281856
            // proxy for path_open !! -1 1 21021328 8 0 267910846n 268435455n 0 21281872
            // proxy for path_open !! -1 1 21021328 8 0 267910846n 268435455n 0 21281856
            // TinyGo
            // proxy for path_open !! 3 1 151536 10 0 0n 0n 0 133972
            // proxy for path_open !! 3 1 151536 8 0 0n 0n 0 133972
            console.log('proxy for', k, '!!', ...args);
            const return_value = orig(...args);
            console.log('return_value for', k, 'is', return_value);
            return return_value;
        };
    });
    const wasmUrl = 'move2kube.wasm';
    const wasmModule = await WebAssembly.instantiateStreaming(fetch(wasmUrl), importObject);
    console.log(wasmModule);
    console.log(wasmModule.instance.exports);
    console.log(wasmModule.instance.exports.memory.buffer);
    // console.log(m.instance.exports._start());
    try {
        wasi.start(wasmModule.instance);
    } catch (e) {
        // console.log(typeof e);
        // console.log(e.exit_code);
        // console.log(Object.entries(e));
        console.log('the wasm module finished with exit code:', e);
    }
};

const add_controls = (rootE) => {
    const div_controls = document.createElement('div');
    div_controls.classList.add('controls');

    // const button_start = document.createElement('button');
    // button_start.textContent = 'start';
    // div_controls.appendChild(button_start);

    const label_input_file = document.createElement('label');
    label_input_file.textContent = 'please select a zip/tar archive containing the directory to be processed:';
    const input_file = document.createElement('input');
    input_file.setAttribute('type', 'file');
    input_file.setAttribute('accept', '.zip,.tar,.tar.gz,.tgz');
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
        reader.readAsArrayBuffer(f);
        try {
            const contents = await get_contents;
            console.log('contents', contents);
            const contentsArr = new Uint8Array(contents);
            start_wasm(rootE, f.name, contentsArr);
        } catch (e) {
            console.error(`failed to read the file '${f.name}' . error:`, e);
        }
    });
    const label_download_output = document.createElement('label');
    label_download_output.textContent = 'click on the button to download "myproject" folder';
    const btn_download = document.createElement('button');
    btn_download.textContent = 'Download "myproject"';
    btn_download.addEventListener("click", () => {
        console.log(FILE_SYSTEM);
        downloadArrayBufferAsBlob(FILE_SYSTEM[3].dir.contents["myproject.zip"].data.buffer);
    })
    label_input_file.appendChild(input_file);
    label_download_output.appendChild(btn_download);
    div_controls.appendChild(label_input_file);
    div_controls.appendChild(document.createElement("br"));
    div_controls.appendChild(label_download_output);
    document.body.appendChild(div_controls);
};

const add_styles = () => {
    const styles = document.createElement('style');
    styles.innerHTML = `
* {
    box-sizing: border-box;
}

body {
    margin: 0;
    min-height: 100vh;
}

.controls {
    padding: 1em;
}
`;
    document.head.appendChild(styles);
};

const main = async () => {
    console.log('main start');
    add_styles();

    // create terminal element
    const rootE = document.createElement('div');
    rootE.id = 'div-root';
    rootE.style.width = '1024px';
    rootE.style.height = '640px';
    // rootE.style.border = '1px solid red';
    document.body.appendChild(rootE);

    add_controls(rootE);

    console.log('main done');
};

main().catch(console.error);
