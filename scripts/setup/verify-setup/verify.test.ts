import { describe, expect, it } from "vitest";

import {
  BOILERPLATE_MODULE,
  LOCALIZATION_TOOL_DIRS,
  SAMPLE_REMOVER_DIR,
  SETUP_SHARED_DIR,
  type VerificationInput,
  collectFailures,
  findCodeownersFailures,
  findLeftoverReferences,
  findLicenseFailures,
  findModuleFailures,
  findRepositoryFailures,
  selfDestructTargets,
} from "./verify";

/** 初期化が完了した状態。個々のケースは、ここから 1 箇所だけ崩す。 */
function localized(): VerificationInput {
  return {
    expected: {
      module: "example-api",
      repository: "example-org/example-api",
      copyrightHolder: "Example Org",
      copyrightYear: "2026",
      codeOwners: "@example-org/tech-leads",
    },
    goMod: "module example-api\n\ngo 1.26.5\n",
    license: "MIT License\n\nCopyright (c) 2026 Example Org\n",
    codeowners: "# 例: * @example-org/old\n* @example-org/tech-leads\n",
    readme: "# example-api\n\nhttps://github.com/example-org/example-api\n",
    openapi: "  termsOfService: https://github.com/example-org/example-api\n",
    boilerplateReferences: [],
  };
}

describe("findModuleFailures", () => {
  describe("正常系", () => {
    it("module 行が指定どおりなら違反にしない", () => {
      expect(findModuleFailures("module example-api\n", "example-api")).toEqual([]);
    });

    it("置換先がボイラープレート名そのものなら残存を違反にしない", () => {
      expect(findModuleFailures(`module ${BOILERPLATE_MODULE}\n`, BOILERPLATE_MODULE)).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("module 行が置換されていなければ違反にする", () => {
      expect(findModuleFailures("module go-boilerplate\n", "example-api")).toHaveLength(2);
    });

    it("module 行以外にボイラープレート名が残っていれば違反にする", () => {
      const failures = findModuleFailures("module example-api\n\n// go-boilerplate\n", "example-api");

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("残っています");
    });

    it("module 行の部分一致を一致と誤認しない", () => {
      expect(findModuleFailures("module example-api-v2\n", "example-api")).toHaveLength(1);
    });

    it("正規表現メタ文字を含むモジュール名も literal として突き合わせる", () => {
      expect(findModuleFailures("module a.b/c\n", "a.b/c")).toEqual([]);
      expect(findModuleFailures("module axb/c\n", "a.b/c")).toHaveLength(1);
    });
  });
});

describe("findLeftoverReferences", () => {
  describe("正常系", () => {
    it("残存が無ければ違反にしない", () => {
      expect(findLeftoverReferences([])).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("残存箇所を列挙して違反にする", () => {
      const failures = findLeftoverReferences(["internal/a.go", "cmd/b.go"]);

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("internal/a.go");
      expect(failures[0]).toContain("cmd/b.go");
    });
  });
});

describe("findLicenseFailures", () => {
  describe("正常系", () => {
    it("指定の権利者と年が入っていれば違反にしない", () => {
      expect(findLicenseFailures("Copyright (c) 2026 Example Org", "Example Org", "2026")).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("権利者が違えば違反にする", () => {
      expect(findLicenseFailures("Copyright (c) 2026 Someone", "Example Org", "2026")).toHaveLength(1);
    });

    it("年が違えば違反にする", () => {
      expect(findLicenseFailures("Copyright (c) 2025 Example Org", "Example Org", "2026")).toHaveLength(1);
    });

    it("著作権表示そのものが無ければ違反にする", () => {
      expect(findLicenseFailures("MIT License", "Example Org", "2026")).toHaveLength(1);
    });
  });
});

describe("findCodeownersFailures", () => {
  describe("正常系", () => {
    it("全ルールが指定所有者なら違反にしない", () => {
      expect(findCodeownersFailures("* @org/team\n/docs @org/team\n", "@org/team")).toEqual([]);
    });

    it("コメント行の例示所有者は対象外にする", () => {
      expect(findCodeownersFailures("# * @org/example\n* @org/team\n", "@org/team")).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("置換され残ったルールを挙げて違反にする", () => {
      const failures = findCodeownersFailures("* @org/team\n/docs @org/old\n", "@org/team");

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("/docs @org/old");
    });

    it("ルールが 1 件も無ければ検査が的を外しているとして違反にする", () => {
      const failures = findCodeownersFailures("# コメントだけ\n\n", "@org/team");

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("的を外しています");
    });
  });
});

describe("findRepositoryFailures", () => {
  describe("正常系", () => {
    it("README と OpenAPI の双方に参照があれば違反にしない", () => {
      expect(findRepositoryFailures("org/repo", "org/repo", "org/repo")).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("README に参照が無ければ違反にする", () => {
      expect(findRepositoryFailures("", "org/repo", "org/repo")).toHaveLength(1);
    });

    it("OpenAPI に参照が無ければ違反にする", () => {
      expect(findRepositoryFailures("org/repo", "", "org/repo")).toHaveLength(1);
    });

    it("双方に無ければ両方を挙げる", () => {
      expect(findRepositoryFailures("", "", "org/repo")).toHaveLength(2);
    });
  });
});

describe("collectFailures", () => {
  describe("正常系", () => {
    it("初期化が完了していれば違反ゼロ", () => {
      expect(collectFailures(localized())).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("モジュール名が未置換なら違反にする", () => {
      const input = { ...localized(), goMod: "module go-boilerplate\n" };

      expect(collectFailures(input).length).toBeGreaterThan(0);
    });

    it("複数の未完了をまとめて返す", () => {
      const input = { ...localized(), license: "MIT License\n", readme: "" };

      expect(collectFailures(input)).toHaveLength(2);
    });
  });
});

describe("selfDestructTargets", () => {
  describe("正常系", () => {
    it("初期化ツール一式と自身を対象にする", () => {
      const targets = selfDestructTargets("verify-setup", true);

      for (const dir of LOCALIZATION_TOOL_DIRS) {
        expect(targets).toContain(dir);
      }
      expect(targets).toContain("verify-setup");
    });

    it("サンプル削除ツールが残っていれば共有モジュールは残す", () => {
      expect(selfDestructTargets("verify-setup", true)).not.toContain(SETUP_SHARED_DIR);
    });

    it("サンプル削除ツールが既に消えていれば共有モジュールも道連れにする", () => {
      expect(selfDestructTargets("verify-setup", false)).toContain(SETUP_SHARED_DIR);
    });

    it("ディレクトリ単位で挙げ、個別ファイルを列挙しない", () => {
      expect(selfDestructTargets("verify-setup", false).some((t) => t.endsWith(".ts"))).toBe(false);
    });
  });
});

describe("LOCALIZATION_TOOL_DIRS", () => {
  describe("正常系", () => {
    it("サンプル削除ツールを初期化ツールに数えない", () => {
      expect(LOCALIZATION_TOOL_DIRS).not.toContain(SAMPLE_REMOVER_DIR);
    });
  });
});
