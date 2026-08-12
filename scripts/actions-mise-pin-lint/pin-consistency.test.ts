import { describe, expect, it } from "vitest";

import { DIGEST_PREFIX_LENGTH, findViolations, type MisePin, readPin } from "./pin-consistency";

const VERSION = "2026.7.12";
const DIGEST = "dad54e0b843908324282b8673f9c0ebc3a4da0c49ad2da309a49bfbc918ba180";

function pin(overrides: Partial<MisePin> = {}): MisePin {
  return {
    version: VERSION,
    digest: DIGEST,
    cacheKey: `mise-\${{ runner.os }}-\${{ runner.arch }}-${VERSION}-${DIGEST.slice(0, DIGEST_PREFIX_LENGTH)}`,
    ...overrides,
  };
}

describe("DIGEST_PREFIX_LENGTH", () => {
  it("キャッシュキーへ埋める digest の桁数を示す", () => {
    expect(DIGEST_PREFIX_LENGTH).toBe(8);
  });
});

describe("readPin", () => {
  it("版 / digest / キャッシュキーを読み取る", () => {
    const source = [
      "        key: mise-Linux-X64-2026.7.12-dad54e0b",
      "      env:",
      `        MISE_VERSION: ${VERSION}`,
      `        MISE_SHA256: ${DIGEST}`,
    ].join("\n");

    expect(readPin(source)).toEqual({
      version: VERSION,
      digest: DIGEST,
      cacheKey: "mise-Linux-X64-2026.7.12-dad54e0b",
    });
  });

  it("読み取れない値を null で返す", () => {
    expect(readPin("name: Setup mise")).toEqual({
      version: null,
      digest: null,
      cacheKey: null,
    });
  });

  it("空の値を null で返す", () => {
    expect(readPin("MISE_VERSION:   \nMISE_SHA256:\nkey: \t")).toEqual({
      version: null,
      digest: null,
      cacheKey: null,
    });
  });
});

describe("findViolations", () => {
  it("揃っていれば違反を報告しない", () => {
    expect(findViolations(pin())).toEqual([]);
  });

  it("キャッシュキーが版を含まなければ落とす", () => {
    expect(findViolations(pin({ cacheKey: "mise-Linux-X64-2026.6.0-dad54e0b" }))).toHaveLength(1);
  });

  it("キャッシュキーが digest の先頭を含まなければ落とす", () => {
    expect(findViolations(pin({ cacheKey: `mise-Linux-X64-${VERSION}-00000000` }))).toHaveLength(1);
  });

  it("読み取れない値をそれぞれ違反として報告する", () => {
    expect(findViolations({ version: null, digest: null, cacheKey: null })).toEqual([
      "MISE_VERSION を読み取れません",
      "MISE_SHA256 を読み取れません",
      "キャッシュの key を読み取れません",
    ]);
  });
});
