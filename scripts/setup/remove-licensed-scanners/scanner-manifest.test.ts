import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";
import { SCANNER_DOMAINS } from "./scanner-manifest";

describe("SCANNER_DOMAINS", () => {
  const read = (file: string): string => fs.readFileSync(path.join(ROOT_DIR, file), "utf8");

  describe("正常系", () => {
    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: 削除対象のパスが実在する",
      (_key, domain) => {
        for (const relativePath of domain.paths) {
          expect(fs.existsSync(path.join(ROOT_DIR, relativePath)), relativePath).toBe(true);
        }
      },
    );

    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: 宣言した README の行が現物に完全一致で存在する",
      (_key, domain) => {
        for (const entry of domain.docBlocks) {
          expect(read(entry.file), `${entry.file}: ${entry.block.trim().slice(0, 60)}`).toContain(
            entry.block,
          );
        }
      },
    );

    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: 宣言した README の語句が現物に完全一致で存在する",
      (_key, domain) => {
        for (const entry of domain.docFragments) {
          expect(read(entry.file), `${entry.file}: ${entry.fragment}`).toContain(entry.fragment);
        }
      },
    );

    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: 宣言した見出しが現物に存在する",
      (_key, domain) => {
        for (const entry of domain.docSections) {
          expect(read(entry.file).split("\n"), `${entry.file}: ${entry.heading}`).toContain(
            entry.heading,
          );
        }
      },
    );

    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: 宣言した lockfile キーが現物に存在する",
      (_key, domain) => {
        const lockfile = read(".github/actions-pin.toml");

        for (const key of domain.pinKeys) {
          expect(lockfile, key).toContain(`"${key}"`);
        }
      },
    );

    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: 宣言した egress セクションが現物に存在する",
      (_key, domain) => {
        const ssot = read(".github/egress.toml");

        for (const job of domain.egressJobs) {
          expect(ssot, job).toContain(`[job."${job}"]`);
        }
      },
    );

    it.each(SCANNER_DOMAINS.map((domain) => [domain.key, domain] as const))(
      "%s: presenceMarker が削除対象に含まれる（撤去後に必ず消え、2 回目実行がスキップされる）",
      (_key, domain) => {
        expect(domain.paths).toContain(domain.presenceMarker);
      },
    );

    it("Bearer は対象に含めない（ELv2 は CI 実行を制約しないため）", () => {
      const paths = SCANNER_DOMAINS.flatMap((domain) => domain.paths);

      expect(paths).not.toContain(".github/workflows/bearer.yaml");
    });

    it("コミット subject が commitlint の type-enum に適合する", () => {
      const types =
        /^(Feat|Fix|Refactor|Perf|Docs|Test|Build|CI|Chore|Style|Revert): .+/;

      for (const domain of SCANNER_DOMAINS) {
        expect(domain.commitSubject, domain.key).toMatch(types);
      }
    });

    it("CodeQL を最後に置く（2 件をまとめて説明する散文を持つため）", () => {
      expect(SCANNER_DOMAINS.at(-1)?.key).toBe("code-ql");
    });
  });
});
