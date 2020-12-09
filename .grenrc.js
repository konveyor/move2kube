let printPreamble = true;
const preamble = `# Changelog
For more documentation and support please visit the website https://konveyor.github.io/move2kube/
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
  "prefix": "Move2Kube ",
  "groupBy":
  {
    "🚀 Features": ["enhancement"],
    "🐛 Bug Fixes": ["bug"]
  },
  "template": {
    "group": printPreambleAndGroupName,
    "issue": ({ name, text, url }) => `- ${name} [${text}](${url})`,
  }
}
