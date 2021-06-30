/*
 *  Copyright IBM Corporation 2020, 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

import url from 'url';
import http from 'http';
import redis from "redis"

const api_endpoint = '/fib';
const usage_instructions = `usage: ${api_endpoint}?n=<your number>\n`;
const client = 'REDIS_URL' in process.env ? redis.createClient(process.env.REDIS_URL) : redis.createClient();
client.on("error", err => console.error(err));

function requestHandler(req, res) {
    const urlobj = url.parse(req.url, true);

    if (urlobj.pathname !== api_endpoint || !('n' in urlobj.query)) {
        res.writeHead(400, { "Content-Type": "application/json" });
        return res.end(JSON.stringify({error:"invalid url",usage_instructions}));
    }

    const n = parseInt(urlobj.query.n, 10);
    if (isNaN(n)) {
        res.writeHead(400, { "Content-Type": "application/json" });
        return res.end(JSON.stringify({error:"n is not a valid number"}));
    }

    client.get(n, (err, ans) => {
        if (err || ans === null) {
            console.log(`CACHE MISS on n = ${n}`);
            ans = fibonacci(n);
            client.set(n, ans, err => {
                if (err === null) console.log("cached", n);
                else console.error("failed to cache the answer for n. error:", err)
            });
        } else {
            console.log(`CACHE HIT for n = ${n} ans is ${ans}`);
        }
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ans}));
    });
}

function fibonacci(n) {
    let a = 0, b = 1, c = 1;
    for(let i = 0; i < n; i++) {
        c = a + b;
        a = b;
        b = c;
    }
    return a;
}

// Main
http.createServer(requestHandler).listen(1234);
