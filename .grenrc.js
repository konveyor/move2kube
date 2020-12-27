let printPreamble = true;
const preamble = `For more documentation and support please visit https://konveyor.io/move2kube/
# Changelog
`;

function printPreambleAndGroupName({ heading }) {
  const line = `\n## ${heading}\n`;
  if (printPreamble) {
    printPreamble = false;
    return preamble + line;
  }
  return line;
}

module.exports = {
  "dataSource": "prs",
  "prefix": "[WIP] Move2Kube ",
  // valid PR types: ['feat', 'fix', 'docs', 'style', 'refactor', 'perf', 'test', 'build', 'ci', 'chore', 'revert']
  "groupBy":
  {
    "🚀 Features": ["enhancement", "feat", "perf"],
    "🐛 Bug Fixes": ["bug", "fix", "revert"],
    "🧹 Maintenance": ["docs", "style", "refactor", "test", "build", "ci", "chore"]
  },
  "template": {
    "group": printPreambleAndGroupName,
    "issue": ({ name, text, url }) => `- ${name} [${text}](${url})`,
  }
}
