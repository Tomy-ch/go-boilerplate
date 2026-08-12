export type MisePin = {
  version: string | null;
  digest: string | null;
  cacheKey: string | null;
};

const VERSION_PATTERN = /^\s*MISE_VERSION:\s*(\S+)\s*$/m;
const DIGEST_PATTERN = /^\s*MISE_SHA256:\s*(\S+)\s*$/m;
const CACHE_KEY_PATTERN = /^\s*key:\s*(\S.*?)\s*$/m;

export const DIGEST_PREFIX_LENGTH = 8;

export function readPin(source: string): MisePin {
  return {
    version: VERSION_PATTERN.exec(source)?.[1] ?? null,
    digest: DIGEST_PATTERN.exec(source)?.[1] ?? null,
    cacheKey: CACHE_KEY_PATTERN.exec(source)?.[1] ?? null,
  };
}

export function findViolations(pin: MisePin): string[] {
  const violations: string[] = [];
  if (pin.version === null) violations.push("MISE_VERSION を読み取れません");
  if (pin.digest === null) violations.push("MISE_SHA256 を読み取れません");
  if (pin.cacheKey === null) violations.push("キャッシュの key を読み取れません");
  if (pin.version === null || pin.digest === null || pin.cacheKey === null) return violations;

  if (!pin.cacheKey.includes(pin.version)) {
    violations.push(`キャッシュキーが版を含んでいません（版 ${pin.version} / キー ${pin.cacheKey}）`);
  }

  const prefix = pin.digest.slice(0, DIGEST_PREFIX_LENGTH);
  if (!pin.cacheKey.includes(prefix)) {
    violations.push(
      `キャッシュキーが digest の先頭 ${DIGEST_PREFIX_LENGTH} 桁を含んでいません（${prefix} / キー ${pin.cacheKey}）`,
    );
  }

  return violations;
}
