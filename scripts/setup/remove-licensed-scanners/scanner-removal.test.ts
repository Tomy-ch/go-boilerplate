import { describe, expect, it } from "vitest";

import {
  MalformedHeadingError,
  MissingDeclarationError,
  isPinKeyReferenced,
  removeEgressSections,
  removeExact,
  removePinEntries,
  removeSection,
  repoOf,
} from "./scanner-removal";

describe("repoOf", () => {
  describe("正常系", () => {
    it("タグを落として owner/repo を返す", () => {
      expect(repoOf("github/codeql-action@v4")).toBe("github/codeql-action");
    });

    it("タグに @ を含んでも最後の @ で割る", () => {
      expect(repoOf("owner/repo@v1@rc")).toBe("owner/repo@v1");
    });
  });

  describe("異常系", () => {
    it("@ が無ければそのまま返す", () => {
      expect(repoOf("owner/repo")).toBe("owner/repo");
    });
  });
});

describe("isPinKeyReferenced", () => {
  describe("正常系", () => {
    it("残るワークフローが参照していれば true", () => {
      expect(
        isPinKeyReferenced({
          key: "SonarSource/sonarqube-scan-action@v8.2.1",
          survivingContents: ["      uses: SonarSource/sonarqube-scan-action@abc # v8.2.1"],
        }),
      ).toBe(true);
    });

    it("どこからも参照が無ければ false", () => {
      expect(
        isPinKeyReferenced({
          key: "SonarSource/sonarqube-scan-action@v8.2.1",
          survivingContents: ["      uses: actions/checkout@abc # v7.0.0"],
        }),
      ).toBe(false);
    });

    it("サブパス付きの uses でも owner/repo で拾う", () => {
      expect(
        isPinKeyReferenced({
          key: "github/codeql-action@v4",
          survivingContents: ["      uses: github/codeql-action/upload-sarif@abc # v4"],
        }),
      ).toBe(true);
    });
  });

  describe("異常系", () => {
    it("残るファイルが 1 件も無ければ false", () => {
      expect(
        isPinKeyReferenced({ key: "SonarSource/sonarqube-scan-action@v8.2.1", survivingContents: [] }),
      ).toBe(false);
    });
  });
});

describe("removePinEntries", () => {
  const lockfile = [
    "# コメント",
    '"actions/checkout@v7.0.0" = "aaa"',
    '"SonarSource/sonarqube-scan-action@v8.2.1" = "bbb"',
    '"github/codeql-action@v4" = "ccc"',
    "",
  ].join("\n");

  describe("正常系", () => {
    it("指定したキーの行だけを落とす", () => {
      const result = removePinEntries(lockfile, ["SonarSource/sonarqube-scan-action@v8.2.1"]);

      expect(result).not.toContain("SonarSource/sonarqube-scan-action");
      expect(result).toContain('"actions/checkout@v7.0.0"');
      expect(result).toContain('"github/codeql-action@v4"');
      expect(result).toContain("# コメント");
    });

    it("キーが空なら本文を変えない", () => {
      expect(removePinEntries(lockfile, [])).toBe(lockfile);
    });
  });

  describe("異常系", () => {
    it("存在しないキーを渡しても何も落とさない", () => {
      expect(removePinEntries(lockfile, ["absent/action@v1"])).toBe(lockfile);
    });
  });
});

describe("removeEgressSections", () => {
  const ssot = [
    "[class.base]",
    'hosts = ["a:443"]',
    "",
    '[job."sonarqube.yaml:sonarqube"]',
    'classes = ["mise"]',
    "extra = [",
    '  "sonarcloud.io:443",',
    "]",
    "",
    '[job."sql-lint.yaml:sql-lint"]',
    'classes = ["mise"]',
    "",
  ].join("\n");

  describe("正常系", () => {
    it("セクションを本文ごと落とし、後続のセクションは残す", () => {
      const result = removeEgressSections(ssot, ["sonarqube.yaml:sonarqube"]);

      expect(result).not.toContain("sonarqube");
      expect(result).toContain('[job."sql-lint.yaml:sql-lint"]');
      expect(result).toContain("[class.base]");
    });

    it("キーが空なら本文を変えない", () => {
      expect(removeEgressSections(ssot, [])).toBe(ssot);
    });
  });

  describe("異常系", () => {
    it("存在しないセクションを渡しても他を巻き込まない", () => {
      expect(removeEgressSections(ssot, ["absent.yaml:absent"])).toBe(ssot);
    });
  });
});

describe("MissingDeclarationError", () => {
  describe("正常系", () => {
    it("どのファイルの何が見つからなかったかをメッセージに含める", () => {
      const error = new MissingDeclarationError("README.md", "語句", "+ `sonarqube.yaml`");

      expect(error.name).toBe("MissingDeclarationError");
      expect(error.message).toContain("README.md");
      expect(error.message).toContain("語句");
      expect(error.message).toContain("+ `sonarqube.yaml`");
    });
  });
});

describe("MalformedHeadingError", () => {
  describe("正常系", () => {
    it("どのファイルのどの宣言が見出しでなかったかをメッセージに含める", () => {
      const error = new MalformedHeadingError("README.md", "Bearer のライセンス");

      expect(error.name).toBe("MalformedHeadingError");
      expect(error.message).toContain("README.md");
      expect(error.message).toContain("Bearer のライセンス");
    });
  });
});

describe("removeExact", () => {
  describe("正常系", () => {
    it("完全一致した箇所だけを取り除く", () => {
      expect(removeExact("a + b + c", " + b", "f.md", "語句")).toBe("a + c");
    });

    it("同じ文字列が複数あっても最初の 1 箇所だけを取り除く", () => {
      expect(removeExact("x,x,", "x,", "f.md", "語句")).toBe("x,");
    });
  });

  describe("異常系", () => {
    it("見つからなければ投げる（README のドリフトを黙って見逃さない）", () => {
      expect(() => removeExact("abc", "zzz", "f.md", "語句")).toThrow(MissingDeclarationError);
    });
  });
});

describe("removeSection", () => {
  const doc = [
    "## 見出し A",
    "",
    "本文 A",
    "",
    "#### 対象",
    "",
    "消える本文",
    "",
    "#### 次",
    "",
    "残る本文",
    "",
  ].join("\n");

  describe("正常系", () => {
    it("見出しと本文を次の見出しの手前まで落とす", () => {
      const result = removeSection(doc, "#### 対象", "f.md");

      expect(result).not.toContain("消える本文");
      expect(result).toContain("#### 次");
      expect(result).toContain("残る本文");
    });

    it("見出し手前の空行を残し、隣り合う 2 節を続けて消しても区切りが尽きない", () => {
      const adjacent = ["本文", "", "#### A", "", "a", "", "#### B", "", "b", "", "#### C", ""].join(
        "\n",
      );
      const result = removeSection(removeSection(adjacent, "#### A", "f.md"), "#### B", "f.md");

      expect(result).toBe(["本文", "", "#### C", ""].join("\n"));
    });

    it("上位レベルの見出しでも境界として止まる", () => {
      const nested = ["#### 対象", "", "消える", "", "## 上位", "", "残る", ""].join("\n");

      expect(removeSection(nested, "#### 対象", "f.md")).toContain("## 上位");
    });

    it("最後のセクションなら本文の終端までを落とす", () => {
      const tail = ["## A", "", "本文", "", "#### 対象", "", "消える", ""].join("\n");
      const result = removeSection(tail, "#### 対象", "f.md");

      expect(result).not.toContain("消える");
      expect(result).toContain("本文");
    });
  });

  describe("異常系", () => {
    it("見出しが無ければ投げる", () => {
      expect(() => removeSection(doc, "#### 無い", "f.md")).toThrow(MissingDeclarationError);
    });

    it("見出しの形をしていない宣言は、同じ行が本文にあっても投げる", () => {
      expect(() => removeSection(doc, "本文 A", "f.md")).toThrow(MalformedHeadingError);
    });

    it("# の後ろに空白が無い宣言も投げる", () => {
      expect(() => removeSection("####対象\n\n消える\n", "####対象", "f.md")).toThrow(
        MalformedHeadingError,
      );
    });
  });
});
