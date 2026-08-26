import { describe, expect, it } from "vitest";

import {
  type EnvLayout,
  agentName,
  checkAgentParity,
  checkCodexSkillStructure,
  checkFrontmatter,
  checkPlatformOnlyAllowlist,
  checkReferences,
  checkSkillParity,
  formatFindings,
  isClaudeAgentDefinition,
  isCodexAgentDefinition,
  isMakefileFragment,
  isRootMakefile,
} from "./rules";

const LAYOUT: EnvLayout = {
  claudeSkills: ".claude/skills",
  claudeAgents: ".claude/agents",
  codexSkills: ".codex/skills",
  codexAgents: ".codex/agents",
};

const ALLOWLIST_REL = "scripts/skill-lint/checks.ts";

function doc(...lines: string[]): string {
  return lines.join("\n");
}

/** 違反の出ない canonical。個々のケースは、ここから 1 箇所だけ崩す。 */
function canonical(name = "demo"): string {
  return doc("---", `name: ${name}`, "description: what it does", "---", "", "# Demo", "", "## Steps");
}

const NO_REFERENCES = {
  makeTargetExists: () => true,
  repoPathExists: () => true,
  configFileExists: () => true,
  asRepoPath: () => null,
};

describe("checkFrontmatter", () => {
  describe("正常系", () => {
    it("必須キーが揃い配置名と一致すれば違反にしない", () => {
      expect(checkFrontmatter("a/SKILL.md", canonical(), "demo")).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("frontmatter が無ければ 1 行目を指して報告する", () => {
      const findings = checkFrontmatter("a/SKILL.md", "# Demo", "demo");

      expect(findings).toHaveLength(1);
      expect(findings[0]).toMatchObject({ file: "a/SKILL.md", line: 1, rule: "frontmatter" });
    });

    it("name が欠けていれば報告する", () => {
      const findings = checkFrontmatter("a/SKILL.md", doc("---", "description: x", "---"), "demo");

      expect(findings.map((f) => f.message)).toContainEqual(expect.stringContaining("`name`"));
    });

    it("description が空でも欠落として報告する", () => {
      const findings = checkFrontmatter("a/SKILL.md", doc("---", "name: demo", "description:", "---"), "demo");

      expect(findings.map((f) => f.message)).toContainEqual(expect.stringContaining("`description`"));
    });

    it("name が配置名と食い違えば両方を挙げて報告する", () => {
      const findings = checkFrontmatter("a/SKILL.md", canonical("other"), "demo");

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("other");
      expect(findings[0].message).toContain("demo");
    });

    it("必須キーの欠落と配置名の不一致を同時に報告する", () => {
      const findings = checkFrontmatter("a/SKILL.md", doc("---", "name: other", "---"), "demo");

      expect(findings).toHaveLength(2);
    });
  });
});

describe("checkSkillParity", () => {
  describe("正常系", () => {
    it("両環境に揃っていれば違反にしない", () => {
      expect(checkSkillParity(["a"], ["a"], new Map(), LAYOUT, ALLOWLIST_REL)).toEqual([]);
    });

    it("例外に登録されたスキルは片側だけでも違反にしない", () => {
      expect(
        checkSkillParity(["a"], [], new Map([["a", "Claude 専用の機構を使うため"]]), LAYOUT, ALLOWLIST_REL),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("Claude 側にしか無いスキルを対応先込みで報告する", () => {
      const findings = checkSkillParity(["a"], [], new Map(), LAYOUT, ALLOWLIST_REL);

      expect(findings).toHaveLength(1);
      expect(findings[0]).toMatchObject({ file: ".claude/skills/a", rule: "cross-env" });
      expect(findings[0].message).toContain(".codex/skills/a");
      expect(findings[0].message).toContain("sync-ai");
    });

    it("Codex 側にしか無いスキルも同じ形で報告する", () => {
      const findings = checkSkillParity([], ["a"], new Map(), LAYOUT, ALLOWLIST_REL);

      expect(findings).toHaveLength(1);
      expect(findings[0].file).toBe(".codex/skills/a");
      expect(findings[0].message).toContain(".claude/skills/a");
    });

    it("両方向の欠落をまとめて報告する", () => {
      expect(checkSkillParity(["a"], ["b"], new Map(), LAYOUT, ALLOWLIST_REL)).toHaveLength(2);
    });
  });
});

describe("checkAgentParity", () => {
  describe("正常系", () => {
    it("両環境に揃っていれば違反にしない", () => {
      expect(checkAgentParity(["a"], ["a"], LAYOUT)).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("Claude 側にしか無いエージェントを toml 名で対応付けて報告する", () => {
      const findings = checkAgentParity(["a"], [], LAYOUT);

      expect(findings).toHaveLength(1);
      expect(findings[0].file).toBe(".claude/agents/a.md");
      expect(findings[0].message).toContain(".codex/agents/a.toml");
    });

    it("Codex 側にしか無いエージェントを md 名で対応付けて報告する", () => {
      const findings = checkAgentParity([], ["a"], LAYOUT);

      expect(findings[0].file).toBe(".codex/agents/a.toml");
      expect(findings[0].message).toContain(".claude/agents/a.md");
    });

    it("例外の仕組みを持たず片側だけは必ず報告する", () => {
      expect(checkAgentParity(["a"], ["b"], LAYOUT)).toHaveLength(2);
    });
  });
});

describe("checkPlatformOnlyAllowlist", () => {
  describe("正常系", () => {
    it("片側だけに在り理由がある登録は違反にしない", () => {
      expect(
        checkPlatformOnlyAllowlist(["a"], [], new Map([["a", "理由"]]), ALLOWLIST_REL),
      ).toEqual([]);
    });

    it("登録が無ければ違反にしない", () => {
      expect(checkPlatformOnlyAllowlist(["a"], ["a"], new Map(), ALLOWLIST_REL)).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("理由が空の登録を報告する", () => {
      const findings = checkPlatformOnlyAllowlist(["a"], [], new Map([["a", "   "]]), ALLOWLIST_REL);

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("理由がありません");
      expect(findings[0].file).toBe(ALLOWLIST_REL);
    });

    it("両環境に揃った登録は削除させる", () => {
      const findings = checkPlatformOnlyAllowlist(["a"], ["a"], new Map([["a", "理由"]]), ALLOWLIST_REL);

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("両環境に存在します");
    });

    it("どちらにも無い登録は削除させる", () => {
      const findings = checkPlatformOnlyAllowlist([], [], new Map([["a", "理由"]]), ALLOWLIST_REL);

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("どちらの環境にも存在しません");
    });

    it("理由が空でかつ両環境に在れば両方を報告する", () => {
      const findings = checkPlatformOnlyAllowlist(["a"], ["a"], new Map([["a", ""]]), ALLOWLIST_REL);

      expect(findings).toHaveLength(2);
    });
  });
});

describe("checkCodexSkillStructure", () => {
  describe("正常系", () => {
    it("SKILL.md と openai.yaml が揃えば違反にしない", () => {
      expect(
        checkCodexSkillStructure(".codex/skills/a", { canonical: canonical(), hasMetadata: true }),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("SKILL.md が無ければ報告する", () => {
      const findings = checkCodexSkillStructure(".codex/skills/a", {
        canonical: null,
        hasMetadata: true,
      });

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("`SKILL.md`");
    });

    it("openai.yaml が無ければ報告する", () => {
      const findings = checkCodexSkillStructure(".codex/skills/a", {
        canonical: canonical(),
        hasMetadata: false,
      });

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("openai.yaml");
    });
  });
});

describe("checkReferences", () => {
  describe("正常系", () => {
    it("実在する参照だけなら違反にしない", () => {
      expect(checkReferences("a/SKILL.md", "`make build` を実行する", NO_REFERENCES)).toEqual([]);
    });

    it("コードフェンスの中は参照として読まない", () => {
      const content = doc("```sh", "make no-such-target", "```");

      expect(
        checkReferences("a/SKILL.md", content, { ...NO_REFERENCES, makeTargetExists: () => false }),
      ).toEqual([]);
    });

    it("ignore ディレクティブのある行は飛ばす", () => {
      const content = "`docs/absent.md` <!-- skill-lint-ignore -->";

      expect(
        checkReferences("a/SKILL.md", content, {
          ...NO_REFERENCES,
          asRepoPath: () => "docs/absent.md",
          repoPathExists: () => false,
        }),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("存在しない make ターゲットを行番号付きで報告する", () => {
      const content = doc("見出し", "", "`make no-such-target` を実行する");

      const findings = checkReferences("a/SKILL.md", content, {
        ...NO_REFERENCES,
        makeTargetExists: () => false,
      });

      expect(findings).toHaveLength(1);
      expect(findings[0]).toMatchObject({ file: "a/SKILL.md", line: 3, rule: "make-ref" });
    });

    it("存在しないパスを報告する", () => {
      const findings = checkReferences("a/SKILL.md", "`docs/absent.md`", {
        ...NO_REFERENCES,
        asRepoPath: () => "docs/absent.md",
        repoPathExists: () => false,
      });

      expect(findings).toHaveLength(1);
      expect(findings[0].rule).toBe("path-ref");
    });

    it("パスとして解決できない設定ファイル名の不在を報告する", () => {
      const findings = checkReferences("a/SKILL.md", "`absent.yaml`", {
        ...NO_REFERENCES,
        configFileExists: () => false,
      });

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("設定ファイル");
    });

    it("パス不在を報告した行では設定ファイル検査へ進まない", () => {
      const findings = checkReferences("a/SKILL.md", "`absent.yaml`", {
        ...NO_REFERENCES,
        asRepoPath: () => "absent.yaml",
        repoPathExists: () => false,
        configFileExists: () => false,
      });

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("存在しないパス");
    });

    it("参照元ディレクトリを解決へ渡す", () => {
      const seen: string[] = [];

      checkReferences(".claude/skills/a/SKILL.md", "`scripts/run.sh`", {
        ...NO_REFERENCES,
        asRepoPath: () => "scripts/run.sh",
        repoPathExists: (_candidate, fromDir) => {
          seen.push(fromDir);
          return true;
        },
      });

      expect(seen).toEqual([".claude/skills/a"]);
    });
  });
});

describe("isRootMakefile", () => {
  describe("正常系", () => {
    it("処理系依存の綴りをいずれも受け入れる", () => {
      expect(["makefile", "Makefile", "GNUmakefile"].map(isRootMakefile)).toEqual([true, true, true]);
    });
  });

  describe("異常系", () => {
    it("makefile を名前に含むだけのファイルは読まない", () => {
      expect(isRootMakefile("makefile.bak")).toBe(false);
      expect(isRootMakefile("MAKEFILE")).toBe(false);
    });
  });
});

describe("isMakefileFragment", () => {
  describe("正常系", () => {
    it("拡張子 .mk を材料として読む", () => {
      expect(isMakefileFragment("lint.mk")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("同じディレクトリの README は読まない", () => {
      expect(isMakefileFragment("README.md")).toBe(false);
    });
  });
});

describe("isClaudeAgentDefinition", () => {
  describe("正常系", () => {
    it("md のエージェント定義を対象にする", () => {
      expect(isClaudeAgentDefinition("reviewer.md")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("対訳は定義として数えない", () => {
    });

    it("md 以外は対象にしない", () => {
      expect(isClaudeAgentDefinition("reviewer.toml")).toBe(false);
    });
  });
});

describe("isCodexAgentDefinition", () => {
  describe("正常系", () => {
    it("toml のエージェント定義を対象にする", () => {
      expect(isCodexAgentDefinition("reviewer.toml")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("md は対象にしない", () => {
      expect(isCodexAgentDefinition("reviewer.md")).toBe(false);
    });
  });
});

describe("agentName", () => {
  describe("正常系", () => {
    it("環境ごとに違う拡張子を落として同じ名前にする", () => {
      expect(agentName("reviewer.md")).toBe("reviewer");
      expect(agentName("reviewer.toml")).toBe("reviewer");
    });

    it("名前の途中のドットは残す", () => {
      expect(agentName("a.b.md")).toBe("a.b");
    });
  });
});

describe("formatFindings", () => {
  describe("正常系", () => {
    it("ファイル見出しと行番号・種別を添えて並べる", () => {
      const text = formatFindings([{ file: "a.md", line: 3, rule: "path-ref", message: "壊れています" }]);

      expect(text).toBe("  a.md\n    :3  [path-ref] 壊れています");
    });

    it("走査順に依らずファイル名と行番号で整列する", () => {
      const text = formatFindings([
        { file: "b.md", line: 1, rule: "x", message: "後" },
        { file: "a.md", line: 9, rule: "x", message: "先の 2 行目" },
        { file: "a.md", line: 2, rule: "x", message: "先の 1 行目" },
      ]);

      expect(text).toBe(
        "  a.md\n    :2  [x] 先の 1 行目\n    :9  [x] 先の 2 行目\n\n  b.md\n    :1  [x] 後",
      );
    });
  });

  describe("異常系", () => {
    it("違反が無ければ空文字を返す", () => {
      expect(formatFindings([])).toBe("");
    });
  });
});
