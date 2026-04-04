#!/usr/bin/env node

const version = process.argv[2];
const type = process.argv[3]; // patch | minor | major

if (!version) {
  console.error("version is required");
  process.exit(1);
}

const v = version.replace(/^v/, "");
let [major, minor, patch] = v.split(".").map(Number);

switch (type) {
  case "patch":
    patch++;
    break;
  case "minor":
    minor++;
    patch = 0;
    break;
  case "major":
    major++;
    minor = 0;
    patch = 0;
    break;
  default:
    console.error("type must be patch | minor | major");
    process.exit(1);
}

console.log(`v${major}.${minor}.${patch}`);
