module.exports = {
  "dataSource": "prs",
  "groupBy":
  {
    "🚀 Features": ["enhancement"],
    "🐛 Bug Fixes": ["bug"]
  },
  "template": {
    "commit": ({ message, url, author, name }) => `- [${message}](${url}) - ${author ? `@${author}` : name}`,
    "issue": "- {{labels}} {{name}} [{{text}}]({{url}})",
    "label": "[**{{label}}**]",
    "noLabel": "closed",
    "group": "\n#### {{heading}}\n",
    "changelogTitle": "# Changelog\n\n",
    "release": "## {{release}} ({{date}})\n{{body}}",
    "releaseSeparator": "\n---\n\n"
  }
}
