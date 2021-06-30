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

module.exports = {
  title_prefix: "[WIP] Move2Kube ",
  // valid PR types: ['feat', 'fix', 'docs', 'style', 'refactor', 'perf', 'test', 'build', 'ci', 'chore', 'revert']
  sections: [
    { title: "🚀 Features", labels: ["enhancement", "feat", "perf"] },
    { title: "🐛 Bug Fixes", labels: ["bug", "fix", "revert"] },
    { title: "🧹 Maintenance", labels: ["docs", "style", "refactor", "test", "build", "ci", "chore"] },
  ],
  header: `For more documentation and support please visit https://move2kube.konveyor.io/
# Changelog`,
  line_template: x => `- ${x.title} [#${x.number}](${x.html_url})`,
}
