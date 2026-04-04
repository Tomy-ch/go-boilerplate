const fs = require("fs");
const yaml = require("js-yaml");

const tools = yaml.load(fs.readFileSync("tools.yaml", "utf8")).tools;

const replacements = [
  {
    file: "docker/tools/Dockerfile",
    rules: [
      {
        regex: /migrate@[\w\.\-]+/,
        value: `migrate@${tools.migrate}`,
      },
      {
        regex: /mockgen@[\w\.\-]+/,
        value: `mockgen@${tools.mockgen}`,
      },
      {
        regex: /oapi-codegen@[\w\.\-]+/,
        value: `oapi-codegen@${tools["oapi-codegen"]}`,
      },
      {
        regex: /sqlc@[\w\.\-]+/,
        value: `sqlc@${tools.sqlc}`,
      },
      {
        regex: /sqlfluff\[jinja\](==[\w\.\-]+)?/,
        value: `sqlfluff[jinja]==${tools.sqlfluff}`,
      },
	  {
        regex: /@redocly\/cli@[\w\.\-]+/,
        value: `@redocly/cli@${tools.redocly}`,
      },
    ],
  },
  {
    file: "docker/server/Dockerfile",
    rules: [
      {
        regex: /air@[\w\.\-]+/,
        value: `air@${tools.air}`,
      },
      {
        regex: /dlv@[\w\.\-]+/,
        value: `dlv@${tools.dlv}`,
      },
      {
        regex: /lefthook@[\w\.\-]+/,
        value: `lefthook@${tools.lefthook}`,
      },
      {
        regex: /golangci-lint@[\w\.\-]+/,
        value: `golangci-lint@${tools["golangci-lint"]}`,
      },
      {
        regex: /golines@[\w\.\-]+/,
        value: `golines@${tools.golines}`,
      },
      {
        regex: /gofumpt@[\w\.\-]+/,
        value: `gofumpt@${tools.gofumpt}`,
      },
    ],
  },
  {
    file: ".makefiles/go/installer.mk",
    rules: [
      {
        regex: /gopls@[\w\.\-]+/,
        value: `gopls@${tools.gopls}`,
      },
      {
        regex: /gotests\/\.\.\.@[\w\.\-]+/,
        value: `gotests/...@${tools.gotests}`,
      },
      {
        regex: /impl@[\w\.\-]+/,
        value: `impl@${tools.impl}`,
      },
      {
        regex: /dlv@[\w\.\-]+/,
        value: `dlv@${tools.dlv}`,
      },
      {
        regex: /lefthook@[\w\.\-]+/,
        value: `lefthook@${tools.lefthook}`,
      },
      {
        regex: /golangci-lint@[\w\.\-]+/,
        value: `golangci-lint@${tools["golangci-lint"]}`,
      },
    ],
  },
  {
    file: ".github/workflows/gen-db-artifacts-check.yaml",
    rules: [
      {
        regex: /sqlc@[\w\.\-]+/,
        value: `sqlc@${tools.sqlc}`,
      },
    ],
  },
  {
    file: ".github/workflows/lint.yaml",
    rules: [
      {
        regex: /golangci-lint@[\w\.\-]+/,
        value: `golangci-lint@${tools["golangci-lint"]}`,
      },
    ],
  },
  {
    file: ".github/workflows/vulnerability-check.yaml",
    rules: [
      {
        regex: /govulncheck@[\w\.\-]+/,
        value: `govulncheck@${tools.govulncheck}`,
      },
    ],
  },
];

for (const target of replacements) {
  if (!fs.existsSync(target.file)) {
    console.warn(`Skip (not found): ${target.file}`);
    continue;
  }

  console.log(`Processing: ${target.file}`);

  let content = fs.readFileSync(target.file, "utf8");

  for (const rule of target.rules) {
    content = content.replace(new RegExp(rule.regex, "g"), rule.value);
  }

  fs.writeFileSync(target.file, content);
}

console.log("All tool versions replaced.");
