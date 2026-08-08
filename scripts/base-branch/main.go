// Package main は、フィーチャーブランチの分岐元となるリリースブランチを解決するツール。
//
//	base-branch    最新のリリースライン（`release/vX.Y.Z`）のブランチ名を標準出力へ 1 行で出す
//
// このリポジトリはリリースのたびにリリースラインが移る運用のため、「どこから切るか」の
// 答えが参照先ごとに食い違う。ローカルの `refs/remotes/origin/HEAD` は clone 時に一度
// 決まったきり `git fetch` では更新されず（更新には `git remote set-head` が要る）、
// GitHub のデフォルトブランチは最新のリリースラインへ移る前の値であることがある。
// どちらも警告を出さずに古い答えを返すので、判断の出所を 1 つに固定する。
//
// 出所は origin の実状態（`git ls-remote`）。ローカルの参照は一切読まない。上の 2 つが
// 陳腐化していても答えは変わらない。
//
// 最新の定義はバージョン番号の数値比較であって、コミット日時ではない。リリースラインの
// 新しさはバージョン番号そのものが表しており、日時は古いラインへの hotfix やベース merge で
// 前後する（実際このリポジトリの release/v2.1.0 は、より新しい release/v2.2.0 を取り込んだ
// merge を tip に持つ）。ブランチを切る scripts/release も同じ数値比較で次版を決めるので、
// 作る側と解決する側の基準が揃う。文字列順を採らないのは v1.10.0 が v1.9.0 より前に並ぶため。
//
// git はホストの認証情報を使うため、ツールランナーではなくホストで実行する
// （scripts/release と同じ扱い）。
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	// remoteName は、実状態を問い合わせるリモート。
	remoteName = "origin"
	// refPrefix は、`git ls-remote --heads` が返す参照の接頭辞。
	refPrefix = "refs/heads/"
	// releasePrefix は、リリースラインのブランチ名の接頭辞。
	releasePrefix = "release/"
	// commandTimeout は、git 1 コマンドあたりの上限。ネットワーク越しのため余裕を持たせる。
	commandTimeout = 60 * time.Second
)

var (
	// releasePattern は、リリースラインとして扱うブランチ名の形式。プレリリースや
	// ビルドメタデータは対象外（scripts/release が作る形式に合わせる）。
	releasePattern = regexp.MustCompile(`^release/v(\d+)\.(\d+)\.(\d+)$`)
	// errNoReleaseBranch は、origin にリリースラインが 1 本も無いことを表す。
	errNoReleaseBranch = xerrors.New("❌ origin に release/vX.Y.Z 形式のブランチがありません")
	// errUnexpectedArgs は、解釈できない引数が渡されたことを表す。
	errUnexpectedArgs = xerrors.New("❌ usage: base-branch（引数は取りません）")
)

// releaseLine は、1 本のリリースライン。
type releaseLine struct {
	name                string
	major, minor, patch int
}

// main はエラーを終了コードへ変換するだけに留め、判断は run が持ちます。
// main は 1:1 の対象外でテストを書けないため、ここに分岐を置くと検査されない
// コードがそのぶん増える。
func main() {
	log.SetFlags(0)

	if err := run(os.Args[1:], lsRemoteReleases, os.Stdout); err != nil {
		log.Fatalf("%v", err)
	}
}

// run は、最新のリリースラインのブランチ名を out へ 1 行で書き出します。
// list は origin の参照一覧の取得手段で、差し替えられるよう引数で受けます。
//
// 出力を装飾しないのは、呼び出し側がコマンド置換でそのまま変数へ受けるためです。
// 進捗やエラーは log（標準エラー）側へ出るので、標準出力に混ざることはありません。
func run(args []string, list func() (string, error), out io.Writer) error {
	fs := flag.NewFlagSet("base-branch", flag.ContinueOnError)

	if err := fs.Parse(args); err != nil {
		// ヘルプ要求は失敗ではないので 0 で終える。usage は flag が既に出力している。
		if xerrors.Is(err, flag.ErrHelp) {
			return nil
		}

		return xerrors.Wrap(err, "failed to parse flags")
	}

	// 引数を黙って捨てると、フラグのつもりで渡された指定が無視されたまま
	// もっともらしいブランチ名が返る。
	if fs.NArg() > 0 {
		return xerrors.Wrap(errUnexpectedArgs, strings.Join(fs.Args(), " "))
	}

	refs, err := list()
	if err != nil {
		return err
	}

	latest, err := latestRelease(refs)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(out, latest.name)

	return err
}

// lsRemoteReleases は、origin のリリースラインの参照一覧を取得します。
func lsRemoteReleases() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", remoteName, refPrefix+releasePrefix+"*")
	// git の失敗理由（認証・名前解決）はそのまま利用者へ見せる。握り潰すと
	// 「リリースラインが無い」との区別が付かなくなる。
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", xerrors.Wrap(err, "git ls-remote --heads "+remoteName)
	}

	return string(out), nil
}

// latestRelease は、`git ls-remote --heads` の出力から最新のリリースラインを選びます。
//
// 解釈できる行が 1 つも無い場合はエラーにします。取得自体は成功しうる（リモートに
// リリースラインがまだ無い、参照の書式が変わった）ため、0 件を「最新は空文字」として
// 返すと、呼び出し側は解決できなかったことに気付かないまま空のベースを使います。
func latestRelease(lsRemoteOutput string) (releaseLine, error) {
	var (
		latest releaseLine
		found  bool
	)

	for line := range strings.Lines(lsRemoteOutput) {
		parsed, ok := parseLine(line)
		if !ok {
			continue
		}

		if !found || parsed.newerThan(latest) {
			latest, found = parsed, true
		}
	}

	if !found {
		return releaseLine{}, errNoReleaseBranch
	}

	return latest, nil
}

// parseLine は、`git ls-remote --heads` の 1 行（`<sha>\t<ref>`）をリリースラインへ変換します。
// リリースラインの書式に合わない行は対象外として false を返します。
func parseLine(line string) (releaseLine, bool) {
	_, ref, ok := strings.Cut(strings.TrimSpace(line), "\t")
	if !ok {
		return releaseLine{}, false
	}

	name, ok := strings.CutPrefix(strings.TrimSpace(ref), refPrefix)
	if !ok {
		return releaseLine{}, false
	}

	m := releasePattern.FindStringSubmatch(name)
	if m == nil {
		return releaseLine{}, false
	}

	major, minor, patch, ok := parseTriple(m[1], m[2], m[3])
	if !ok {
		return releaseLine{}, false
	}

	return releaseLine{name: name, major: major, minor: minor, patch: patch}, true
}

// parseTriple は、バージョンの 3 数値を整数へ変換します。桁が int に収まらない場合は
// false を返します。0 として扱うと、あり得ない大きさの版が最古として紛れ込みます。
func parseTriple(rawMajor, rawMinor, rawPatch string) (int, int, int, bool) {
	major, err := strconv.Atoi(rawMajor)
	if err != nil {
		return 0, 0, 0, false
	}

	minor, err := strconv.Atoi(rawMinor)
	if err != nil {
		return 0, 0, 0, false
	}

	patch, err := strconv.Atoi(rawPatch)
	if err != nil {
		return 0, 0, 0, false
	}

	return major, minor, patch, true
}

// newerThan は、レシーバが other より新しいリリースラインかを返します。
func (r releaseLine) newerThan(other releaseLine) bool {
	if r.major != other.major {
		return r.major > other.major
	}

	if r.minor != other.minor {
		return r.minor > other.minor
	}

	return r.patch > other.patch
}
