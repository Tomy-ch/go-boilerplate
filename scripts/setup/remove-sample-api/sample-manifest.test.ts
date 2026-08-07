import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";
import { stripSampleMarkers } from "./sample-api";
import { BUILD_STEPS, MARKER_FILES, SAMPLE_DOMAINS } from "./sample-manifest";

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

describe("MARKER_FILES", () => {
  describe("正常系", () => {
    it("すべて実在する", () => {
      for (const relativePath of MARKER_FILES) {
        expect(fs.existsSync(path.join(ROOT_DIR, relativePath)), relativePath).toBe(true);
      }
    });

    // マーカーが別ファイルへ移った後も登録だけが残ると、除去は 0 行で静かに成功する。
    // 実在確認では通ってしまうため、除去される行があることまで確かめないと
    // 「共有ファイルの手当てが不要になった」と「必要な手当てを書き忘れた」を見分けられない。
    it("登録したファイルが除去される行を実際に持つ", () => {
      for (const relativePath of MARKER_FILES) {
        const content = fs.readFileSync(path.join(ROOT_DIR, relativePath), "utf8");

        expect(stripSampleMarkers(content).removed, relativePath).toBeGreaterThan(0);
      }
    });
  });

  describe("異常系", () => {
    it("同じファイルを二重に登録していない", () => {
      expect(MARKER_FILES).toHaveLength(new Set(MARKER_FILES).size);
    });

    // 削除されるファイルをマーカー除去の対象にも入れると、消えるファイルを書き換える
    // 指定になる。どちらの手当てが要るのかを宣言の時点で取り違えている合図。
    it("削除対象のパスと重ならない", () => {
      const deleted = MARKER_FILES.filter((marker) =>
        registeredPaths.some((registered) => marker === registered || marker.startsWith(`${registered}/`)),
      );

      expect(deleted).toEqual([]);
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
