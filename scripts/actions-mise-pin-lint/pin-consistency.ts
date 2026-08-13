export type MisePin = {
  version: string | null;
  digest: string | null;
  cacheKey: string | null;
};

export const DIGEST_PREFIX_LENGTH = 8;

function readYamlValue(source: string, key: string): string | null {
  const prefix = `${key}:`;
  for (const line of source.split("\n")) {
    const value = line.trim();
    if (!value.startsWith(prefix)) continue;

    const parsed = value.slice(prefix.length).trim();
    return parsed === "" ? null : parsed;
  }
  return null;
}

export function readPin(source: string): MisePin {
  return {
    version: readYamlValue(source, "MISE_VERSION"),
    digest: readYamlValue(source, "MISE_SHA256"),
    cacheKey: readYamlValue(source, "key"),
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
