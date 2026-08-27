/**
 * egress SSOT（`.github/egress.toml`）を書き換える判定。
 *
 * @remarks
 * 撤去ツールをまたいでここに置いています。SSOT の宣言は workflow のジョブと 1:1 で対応し、
 * 自消滅するツールは自分の workflow を消すときこの宣言も道連れにしなければならないため、
 * 先に消えたツールのディレクトリに置くと、後から走るツールが同じ判定を失います。
 */

/** egress SSOT から `[job."<key>"]` セクションを、次のセクション見出しまで本文ごと落とす。 */
export function removeEgressSections(content: string, jobKeys: readonly string[]): string {
  if (jobKeys.length === 0) {
    return content;
  }

  const targets = new Set(jobKeys.map((key) => `[job."${key}"]`));
  const kept: string[] = [];
  let dropping = false;

  for (const line of content.split("\n")) {
    if (/^\[(class|job)\./.test(line)) {
      dropping = targets.has(line.trim());
    }

    if (!dropping) {
      kept.push(line);
    }
  }

  return kept.join("\n");
}
