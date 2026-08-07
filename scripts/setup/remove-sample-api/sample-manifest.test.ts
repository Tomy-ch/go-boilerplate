import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";
import { containsSampleMarker, isScanTarget } from "./sample-api";
import {
  BUILD_STEPS,
  EXCLUDED_DIRECTORIES,
  EXCLUDED_PATH_PREFIXES,
  MARKER_LITERAL_FILES,
  SAMPLE_DOMAINS,
} from "./sample-manifest";

const registeredPaths = Object.values(SAMPLE_DOMAINS).flatMap((domain) => domain.paths);

describe("SAMPLE_DOMAINS", () => {
  describe("正常系", () => {
    it("すべてのドメインが説明と 1 件以上のパスを持つ", () => {
      for (const [name, domain] of Object.entries(SAMPLE_DOMAINS)) {
        expect(domain.description, name).not.toBe("");
        expect(domain.paths.length, name).toBeGreaterThan(0);
      }
    });

    // 実装の移動や統合で対象が消えても、削除側は「既に無い」を黙って飛ばすだけなので
    // 宣言だけが古いまま残る。残った宣言は次に似た構成を足す人にとって現状の写しに
    // 見えるうえ、生成物の行なら「生成すれば現れる」のか「生成元ごと消えた」のかを
    // 宣言から判別できなくなる。
    it("登録パスがすべて実在する", () => {
      for (const relativePath of registeredPaths) {
        expect(fs.existsSync(path.join(ROOT_DIR, relativePath)), relativePath).toBe(true);
      }
    });
  });

  describe("異常系", () => {
    // remove-sample-api の assertWithinRoot はここが破れたときの最後の砦であり、
    // 宣言の側でも同じ条件を保つ。空文字はルート自体、`..` はルート外を指す。
    it("ルート外やルート自体を指すパスを持たない", () => {
      for (const relativePath of registeredPaths) {
        expect(relativePath, relativePath).not.toBe("");
        expect(path.isAbsolute(relativePath), relativePath).toBe(false);
        expect(relativePath.split("/"), relativePath).not.toContain("..");
      }
    });

    // 重複していると削除報告の件数が実態より多く出る。宣言の唯一性が崩れている合図でもある。
    it("同じパスを二重に登録していない", () => {
      expect(registeredPaths).toHaveLength(new Set(registeredPaths).size);
    });
  });
});

describe("sampleTooling", () => {
  describe("正常系", () => {
    // 削除ツールが自分自身を登録し損ねると、サンプルを消したリポジトリに削除ツールだけが
    // 残る。配置を変えたときに真っ先に腐るのがここ。
    it("削除ツール自身のディレクトリを登録している", () => {
      expect([...SAMPLE_DOMAINS.sampleTooling.paths]).toEqual(["scripts/setup/remove-sample-api"]);
    });

    // ディレクトリで登録する以上、そのディレクトリが本当にこのツールの実体かどうかが
    // 唯一の担保になる。別の場所へ移してここを直し忘れると、消し漏れではなく誤爆になる。
    it("登録したディレクトリが実際にこのツールの入口と manifest を含む", () => {
      const registered = path.join(ROOT_DIR, SAMPLE_DOMAINS.sampleTooling.paths[0]);

      expect(fs.existsSync(path.join(registered, "index.ts"))).toBe(true);
      expect(fs.existsSync(path.join(registered, "sample-manifest.ts"))).toBe(true);
    });
  });
});

describe("isScanTarget", () => {
  describe("正常系", () => {
    it("除外に当たらないパスを対象にする", () => {
      expect(isScanTarget("internal/di/module/job.go")).toBe(true);
      expect(isScanTarget("docs/rules.md")).toBe(true);
    });

    it("Windows 形式の区切りでも除外を判定する", () => {
      expect(isScanTarget("docs\\portal\\guides\\index.md")).toBe(false);
    });
  });

  describe("異常系", () => {
    // 生成物はマーカーを持っていても再生成で戻るため、除去の対象にしてはいけない。
    it("生成物の接頭辞を対象から外す", () => {
      for (const prefix of EXCLUDED_PATH_PREFIXES) {
        expect(isScanTarget(`${prefix}anything.md`), prefix).toBe(false);
      }
    });

    it("接頭辞が途中まで一致するだけのパスは外さない", () => {
      expect(isScanTarget("docs/portal/manifest.yaml")).toBe(true);
    });
  });
});

describe("EXCLUDED_DIRECTORIES", () => {
  describe("正常系", () => {
    // 依存の取得物を走査すると、他人のコードのコメントを誤って落としうる。
    it("依存の取得物と VCS の内部を挙げている", () => {
      expect(EXCLUDED_DIRECTORIES.has("node_modules")).toBe(true);
      expect(EXCLUDED_DIRECTORIES.has("vendor")).toBe(true);
      expect(EXCLUDED_DIRECTORIES.has(".git")).toBe(true);
    });
  });

  describe("異常系", () => {
    // マーカーを持つ本文が実際に居るディレクトリを外すと、除去が静かに素通りする。
    it("本文の居るディレクトリを外していない", () => {
      expect(EXCLUDED_DIRECTORIES.has("internal")).toBe(false);
      expect(EXCLUDED_DIRECTORIES.has("docs")).toBe(false);
    });
  });
});

describe("MARKER_LITERAL_FILES", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const relativePath of MARKER_LITERAL_FILES) {
        expect(fs.existsSync(path.join(ROOT_DIR, relativePath)), relativePath).toBe(true);
      }
    });

    // 除去を止める宣言なので、対象がもうマーカー文字列を持たないなら宣言のほうが古い。
    // 放置すると「なぜ除去されないのか」が誰にも分からないファイルが増える。
    it("挙げた対象が実際にマーカー文字列を持つ", () => {
      for (const relativePath of MARKER_LITERAL_FILES) {
        const content = fs.readFileSync(path.join(ROOT_DIR, relativePath), "utf8");

        expect(containsSampleMarker(content), relativePath).toBe(true);
      }
    });

    // 宣言したファイルが本当に走査から外れることを、判定側と突き合わせて固定する。
    it("挙げた対象が走査から外れる", () => {
      for (const relativePath of MARKER_LITERAL_FILES) {
        expect(isScanTarget(relativePath), relativePath).toBe(false);
      }
    });
  });

  describe("異常系", () => {
    it("同じファイルを二重に宣言していない", () => {
      expect(MARKER_LITERAL_FILES).toHaveLength(new Set(MARKER_LITERAL_FILES).size);
    });

    // 削除されるパスを除去対象から外しても意味が無く、「マーカーではない」という
    // 宣言だけが残って読み手を誤らせる。どちらの手当てが要るのかの取り違えの合図。
    it("削除対象のパスと重ならない", () => {
      const overlapping = MARKER_LITERAL_FILES.filter((literal) =>
        registeredPaths.some(
          (registered) => literal === registered || literal.startsWith(`${registered}/`),
        ),
      );

      expect(overlapping).toEqual([]);
    });
  });
});

describe("BUILD_STEPS", () => {
  describe("正常系", () => {
    // 生成より先に整形・検査を走らせると、削除直後の壊れた生成物を整形して lint に
    // かける順序になり、原因の分からない失敗になる。
    it("再生成を整形・検査より前に置く", () => {
      const lastGen = Math.max(BUILD_STEPS.indexOf("gen-api"), BUILD_STEPS.indexOf("gen-query"));

      expect(lastGen).toBeGreaterThanOrEqual(0);
      expect(BUILD_STEPS.indexOf("fix")).toBeGreaterThan(lastGen);
      expect(BUILD_STEPS.indexOf("lint")).toBeGreaterThan(BUILD_STEPS.indexOf("fix"));
    });
  });
});
