import { describe, expect, it } from "vitest";

import { replaceCopyright } from "./license-copyright";

const LICENSE = "MIT License\n\nCopyright (c) 2026 Tomy-ch\n\nPermission is hereby granted...\n";

describe("replaceCopyright", () => {
  describe("正常系", () => {
    it("著作権表示の年と権利者を差し替える", () => {
      expect(replaceCopyright(LICENSE, "Example Inc.", "2030")).toBe(
        "MIT License\n\nCopyright (c) 2030 Example Inc.\n\nPermission is hereby granted...\n",
      );
    });

    it("最初の著作権表示だけを置き換える", () => {
      const content = "Copyright (c) 2026 A\nCopyright (c) 2026 B\n";

      expect(replaceCopyright(content, "C", "2030")).toBe(
        "Copyright (c) 2030 C\nCopyright (c) 2026 B\n",
      );
    });
  });

  describe("異常系", () => {
    // 行頭固定を外すと、ライセンス本文中の "... Copyright (c) ..." に言及する散文
    // （MIT 本文の再頒布条項）まで書き換えてライセンス条文を壊す。
    it("行頭以外の Copyright 表記を対象にしない", () => {
      const content =
        "The above Copyright (c) notice shall be included.\n\nCopyright (c) 2026 Tomy-ch\n";

      expect(replaceCopyright(content, "Example Inc.", "2030")).toBe(
        "The above Copyright (c) notice shall be included.\n\nCopyright (c) 2030 Example Inc.\n",
      );
    });

    // 文字列置換だと `$&` が行全体に展開され、権利者名が二重になった LICENSE が残る。
    it("置換パターンに見える権利者名をそのまま書き込む", () => {
      expect(replaceCopyright("Copyright (c) 2026 old\n", "A$&B", "2030")).toBe(
        "Copyright (c) 2030 A$&B\n",
      );
    });

    // 表示が見つからないまま黙って成功すると、他人名義の LICENSE が残ったまま公開される。
    it("著作権表示が無ければ throw する", () => {
      expect(() => replaceCopyright("MIT License\n", "Example Inc.", "2030")).toThrow(
        "著作権表示が見つかりませんでした",
      );
    });

    it("表記ゆれ（Copyright ©）を著作権表示と認めない", () => {
      expect(() => replaceCopyright("Copyright © 2026 Tomy-ch\n", "Example Inc.", "2030")).toThrow(
        "著作権表示が見つかりませんでした",
      );
    });
  });
});
