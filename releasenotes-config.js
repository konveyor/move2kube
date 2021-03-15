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
