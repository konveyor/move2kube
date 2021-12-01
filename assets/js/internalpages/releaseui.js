"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
const semver = __importStar(require("semver"));
const rest_1 = require("@octokit/rest");
const release_info_url = 'https://raw.githubusercontent.com/konveyor/move2kube/gh-pages/_data/releaseinfo.json';
const owner = 'konveyor';
const repo = 'move2kube';
const owner_repos = [
    ['konveyor', 'move2kube'],
    ['konveyor', 'move2kube-api'],
    ['konveyor', 'move2kube-ui'],
    ['konveyor', 'move2kube-operator'],
    ['konveyor', 'move2kube-tests'],
    ['konveyor', 'move2kube-demos'],
    ['konveyor', 'move2kube-transformers'],
];
let PA_TOKEN = '';
function on_document_ready(fn) {
    // see if DOM is already available
    if (document.readyState === "complete" || document.readyState === "interactive") {
        // call on next available tick
        setTimeout(fn, 1);
    }
    else {
        document.addEventListener("DOMContentLoaded", fn);
    }
}
function get_major_minor_patch(v) {
    const x = semver.parse(v);
    return x !== null ? `${x.major}.${x.minor}.${x.patch}` : "";
}
function get_next_alpha_release(data) {
    // the commit to use for the release is always the top commit on main/master
    if (data.next_next.prerelease !== null) {
        return { tag: "v" + semver.inc(data.next_next.prerelease, "prerelease"), prev_tag: data.next_next.prerelease, commit_ref: 'main' };
    }
    if (data.next.prerelease !== null) {
        if (semver.prerelease(data.next.prerelease)[0] === "alpha") {
            return { tag: "v" + semver.inc(data.next.prerelease, "prerelease"), prev_tag: data.next.prerelease, commit_ref: 'main' };
        }
        else {
            return {
                tag: "v" + semver.inc(get_major_minor_patch(data.next.prerelease), "minor") + "-alpha.0",
                prev_tag: "v" + get_major_minor_patch(data.next.prerelease) + "-beta.0",
                commit_ref: 'main',
            };
        }
    }
    return { tag: "v" + semver.inc(data.current.release, "minor") + "-alpha.0", prev_tag: data.current.release + "-beta.0", commit_ref: 'main' };
}
function get_next_beta_release(data) {
    if (data.next.prerelease !== null) {
        if (semver.prerelease(data.next.prerelease)[0] === "alpha") {
            return { tag: "v" + get_major_minor_patch(data.next.prerelease) + "-beta.0", prev_tag: data.current.release + "-beta.0", commit_ref: 'main' };
        }
        else if (semver.prerelease(data.next.prerelease)[0] === "beta") {
            const obj = semver.parse(data.next.prerelease);
            const branch_name = `release-${obj.major}.${obj.minor}`;
            return { tag: "v" + semver.inc(data.next.prerelease, "prerelease"), prev_tag: data.next.prerelease, commit_ref: branch_name };
        }
        else {
            return { error: `cannot do beta release without releasing v${get_major_minor_patch(data.next.prerelease)} first`, tag: '', prev_tag: '', commit_ref: '' };
        }
    }
    return { error: `cannot do beta release. do alpha first`, tag: '', prev_tag: '', commit_ref: '' };
}
function get_next_patch_rc_release(data) {
    const obj = semver.parse(data.current.release);
    const branch_name = `release-${obj.major}.${obj.minor}`;
    if (semver.gt(data.current.prerelease, data.current.release)) {
        return { tag: "v" + semver.inc(data.current.prerelease, "prerelease"), prev_tag: data.current.prerelease, commit_ref: branch_name };
    }
    return { tag: "v" + semver.inc(data.current.release, "patch") + "-rc.0", prev_tag: data.current.release, commit_ref: branch_name };
}
function get_next_non_patch_rc_release(data) {
    if (data.next.prerelease === null) {
        return { error: "there are no prereleases for next version yet. start with alpha first", tag: '', prev_tag: '', commit_ref: '' };
    }
    const obj = semver.parse(data.next.prerelease);
    const branch_name = `release-${obj.major}.${obj.minor}`;
    if (semver.prerelease(data.next.prerelease)[0] === "rc") {
        return { tag: "v" + semver.inc(data.next.prerelease, "prerelease"), prev_tag: data.next.prerelease, commit_ref: branch_name };
    }
    else if (semver.prerelease(data.next.prerelease)[0] === "beta") {
        return { tag: "v" + get_major_minor_patch(data.next.prerelease) + "-rc.0", prev_tag: data.current.release + "-beta.0", commit_ref: branch_name };
    }
    return { error: "cannot go from alpha to rc. release beta first.", tag: '', prev_tag: '', commit_ref: '' };
}
function get_next_patch_release(data) {
    if (semver.gt(data.current.prerelease, data.current.release)) {
        const obj = semver.parse(data.current.release);
        const branch_name = `release-${obj.major}.${obj.minor}`;
        return { tag: "v" + get_major_minor_patch(data.current.prerelease), prev_tag: data.current.release, commit_ref: branch_name };
    }
    return { error: "cannot do patch release. do a rc release first.", tag: '', prev_tag: '', commit_ref: '' };
}
function get_next_non_patch_release(data) {
    if (data.next.prerelease === null) {
        return { error: "there are no prereleases for next version yet. start with alpha first", tag: '', prev_tag: '', commit_ref: '' };
    }
    if (semver.prerelease(data.next.prerelease)[0] === "rc") {
        const obj = semver.parse(data.next.prerelease);
        const branch_name = `release-${obj.major}.${obj.minor}`;
        return { tag: "v" + get_major_minor_patch(data.next.prerelease), prev_tag: data.current.release + "-beta.0", commit_ref: branch_name };
    }
    else if (semver.prerelease(data.next.prerelease)[0] === "beta") {
        return { error: "cannot go from beta to release. do a rc release first.", tag: '', prev_tag: '', commit_ref: '' };
    }
    return { error: "cannot go from alpha to release. do a beta and rc release first.", tag: '', prev_tag: '', commit_ref: '' };
}
function publish_releases(owner_repo_ids) {
    return __awaiter(this, void 0, void 0, function* () {
        const workflow_filename = 'publish.yml';
        const branch_to_run_on = 'main';
        try {
            if (!PA_TOKEN) {
                return alert('the personal access token is invalid.');
            }
            const octokit = new rest_1.Octokit({ auth: PA_TOKEN });
            const resp = yield octokit.actions.createWorkflowDispatch({
                owner,
                repo,
                workflow_id: workflow_filename,
                ref: branch_to_run_on,
                inputs: { owner_repo_ids: JSON.stringify(owner_repo_ids) },
            });
            console.log(resp);
            document.querySelector('#publish-release-release-drafts').classList.add('hidden');
            const ele = document.querySelector('#publish-release-result-success');
            ele.textContent = `Success!! Status: ${resp.status}`;
            ele.classList.remove('hidden');
            document.querySelector('#publish-release-result').classList.remove('hidden');
        }
        catch (err) {
            console.error(err);
            document.querySelector('#publish-release-release-drafts').classList.add('hidden');
            const ele = document.querySelector('#publish-release-result-error');
            ele.textContent = err;
            ele.classList.remove('hidden');
            document.querySelector('#publish-release-result').classList.remove('hidden');
        }
    });
}
function delete_releases(owner_repo_ids) {
    return __awaiter(this, void 0, void 0, function* () {
        const octokit = new rest_1.Octokit({ auth: PA_TOKEN });
        try {
            const promises = owner_repo_ids.map(owner_repo_id => octokit.repos.deleteRelease(owner_repo_id));
            yield Promise.all(promises);
            document.querySelector('#publish-release-release-drafts').classList.add('hidden');
            const ele = document.querySelector('#publish-release-result-success');
            ele.textContent = `Success!!`;
            ele.classList.remove('hidden');
            document.querySelector('#publish-release-result').classList.remove('hidden');
        }
        catch (err) {
            console.error(err);
            document.querySelector('#publish-release-release-drafts').classList.add('hidden');
            const ele = document.querySelector('#publish-release-result-error');
            ele.textContent = err;
            ele.classList.remove('hidden');
            document.querySelector('#publish-release-result').classList.remove('hidden');
        }
    });
}
function update_release_drafts() {
    return __awaiter(this, void 0, void 0, function* () {
        if (!PA_TOKEN) {
            return alert('the personal access token is invalid.');
        }
        const octokit = new rest_1.Octokit({ auth: PA_TOKEN });
        const helper = (owner, repo) => __awaiter(this, void 0, void 0, function* () {
            const response = yield octokit.repos.listReleases({ owner, repo });
            return { owner, repo, response };
        });
        const promises = owner_repos.map(([owner, repo]) => helper(owner, repo));
        const owner_repo_responses = yield Promise.all(promises);
        const drafts = [];
        for (const { owner, repo, response } of owner_repo_responses) {
            for (const release of response.data) {
                if (!release.draft) {
                    continue;
                }
                drafts.push({ owner, repo, release });
            }
        }
        const drafts_grouped_by_tag = {};
        for (const draft of drafts) {
            const tag = draft.release.tag_name;
            if (!(tag in drafts_grouped_by_tag)) {
                drafts_grouped_by_tag[tag] = [];
            }
            drafts_grouped_by_tag[tag].push(draft);
        }
        const release_drafts_el = document.querySelector('#publish-release-release-drafts');
        release_drafts_el.innerHTML = '';
        const entries = Object.entries(drafts_grouped_by_tag);
        for (const [tag, drafts] of entries) {
            const tag_el = document.createElement('div');
            tag_el.classList.add('release-draft');
            const tag_header = document.createElement('div');
            const tag_header_h3 = document.createElement('h3');
            tag_header_h3.textContent = tag;
            tag_header.appendChild(tag_header_h3);
            const owner_repo_ids = drafts.map(draft => ({ owner: draft.owner, repo: draft.repo, release_id: draft.release.id }));
            const tag_header_publish_button = document.createElement('button');
            tag_header_publish_button.textContent = 'Publish';
            tag_header_publish_button.classList.add('btn', 'btn-primary');
            if (drafts.some(draft => draft.release.name.startsWith('[WIP] '))) {
                tag_header_publish_button.setAttribute('disabled', 'true');
                tag_header_h3.textContent += ' [Work In Progress]';
            }
            else {
                tag_header_publish_button.addEventListener('click', () => publish_releases(owner_repo_ids));
            }
            tag_header.appendChild(tag_header_publish_button);
            const tag_header_delete_button = document.createElement('button');
            tag_header_delete_button.textContent = 'Delete';
            tag_header_delete_button.classList.add('btn', 'btn-red');
            tag_header_delete_button.addEventListener('click', () => delete_releases(owner_repo_ids));
            tag_header.appendChild(tag_header_delete_button);
            tag_el.appendChild(tag_header);
            for (let draft of drafts) {
                const draft_el = document.createElement('a');
                draft_el.textContent = `${draft.owner}/${draft.repo} ${draft.release.name}`;
                draft_el.href = draft.release.html_url;
                tag_el.appendChild(draft_el);
            }
            release_drafts_el.appendChild(tag_el);
        }
    });
}
function create_release_draft(release) {
    return __awaiter(this, void 0, void 0, function* () {
        const workflow_filename = 'release.yml';
        const branch_to_run_on = 'main';
        try {
            if (!PA_TOKEN) {
                return alert('the personal access token is invalid.');
            }
            const octokit = new rest_1.Octokit({ auth: PA_TOKEN });
            const resp = yield octokit.actions.createWorkflowDispatch({
                owner,
                repo,
                workflow_id: workflow_filename,
                ref: branch_to_run_on,
                inputs: {
                    tag: release.tag,
                    prev_tag: release.prev_tag,
                    commit_ref: release.commit_ref,
                },
            });
            console.log(resp);
            document.querySelector('#create-release-release-types').classList.add('hidden');
            const ele = document.querySelector('#create-release-result-success');
            ele.textContent = `Success!! Status: ${resp.status}`;
            ele.classList.remove('hidden');
            document.querySelector('#create-release-result').classList.remove('hidden');
        }
        catch (err) {
            console.error(err);
            document.querySelector('#create-release-release-types').classList.add('hidden');
            const ele = document.querySelector('#create-release-result-error');
            ele.textContent = err;
            ele.classList.remove('hidden');
            document.querySelector('#create-release-result').classList.remove('hidden');
        }
    });
}
function add_option(elem, release, message) {
    if ('error' in release) {
        return console.error(`cannot add release ${message} because of error: ${release.error}`);
    }
    const new_button = document.createElement('button');
    new_button.innerHTML = `${message}<br/>tag: ${release.tag}<br/>commit: ${release.commit_ref}<br/>prev: ${release.prev_tag}`;
    new_button.classList.add('btn', 'btn-blue');
    new_button.addEventListener('click', () => create_release_draft(release));
    elem.appendChild(new_button);
}
function setup() {
    return __awaiter(this, void 0, void 0, function* () {
        const pa_token_input_el = document.querySelector('#pa-token-input');
        const pa_token_button_el = document.querySelector('#pa-token-button');
        pa_token_input_el.addEventListener("keyup", e => {
            if (e.key === 'Enter') {
                e.preventDefault();
                pa_token_button_el.click();
            }
        });
        const pa_token_section_el = document.querySelector('#pa-token-section');
        const create_release_section_el = document.querySelector('#create-release-section');
        const publish_release_section_el = document.querySelector('#publish-release-section');
        pa_token_button_el.addEventListener('click', () => __awaiter(this, void 0, void 0, function* () {
            const token = pa_token_input_el.value;
            if (token.length !== 40) {
                return alert(`Personal access token is invalid. Token length should be 40. got: ${token.length}`);
            }
            PA_TOKEN = token;
            try {
                yield update_release_drafts();
            }
            catch (err) {
                return alert(err);
            }
            pa_token_section_el.classList.add('hidden');
            create_release_section_el.classList.remove('hidden');
            publish_release_section_el.classList.remove('hidden');
        }));
        // create section
        const resp = yield fetch(release_info_url);
        const release_info = yield resp.json();
        const create_release_release_types_el = document.querySelector('#create-release-release-types');
        add_option(create_release_release_types_el, get_next_alpha_release(release_info), 'alpha');
        add_option(create_release_release_types_el, get_next_beta_release(release_info), 'beta');
        add_option(create_release_release_types_el, get_next_patch_rc_release(release_info), 'patch rc');
        add_option(create_release_release_types_el, get_next_non_patch_rc_release(release_info), 'non patch rc');
        add_option(create_release_release_types_el, get_next_patch_release(release_info), 'patch release');
        add_option(create_release_release_types_el, get_next_non_patch_release(release_info), 'non patch release');
    });
}
function main() {
    return __awaiter(this, void 0, void 0, function* () {
        yield setup();
    });
}
on_document_ready(() => main().catch(console.error));
//# sourceMappingURL=releaseui.js.map