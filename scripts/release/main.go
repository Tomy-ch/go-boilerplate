// Package main は、リリースタグとリリースブランチを作成するツール。
//
//	tag    -bump <patch|minor|major>                     production HEAD にタグを打ち、GitHub Release を作る
//	branch -bump <patch|minor|major> -prefix <接頭辞>     次バージョンのブランチを切り、デフォルトブランチに設定する
//
// どちらも `git tag` の最新セマンティックバージョンを起点に次バージョンを決める。
//
// 実体が make のレシピではなくここに在るのは、いずれも取り消しの効かない操作
// （タグの push / GitHub Release の作成 / デフォルトブランチの切り替え）を含み、
// 分岐を実地で確かめようとすると本当にリリースするしかないため。手順の組み立てと
// 中止条件を純粋関数へ寄せて、テストで固定する。
//
// git / gh はホストの認証情報を使うため、ツールランナーではなくホストで実行する
// （cmd/db-slot と同じ扱い）。
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	// releaseNoteDir は、タグ作成時に読むリリースノートの置き場所。
	releaseNoteDir = ".github/release"
	// minArgs は、プログラム名 + サブコマンドの最小引数数。
	minArgs = 2
	// commandTimeout は、git / gh 1 コマンドあたりの上限。push や Release 作成は
	// ネットワーク越しのため、対話が無い前提で余裕を持たせる。
	commandTimeout = 120 * time.Second
)

var (
	// semverPattern は、リリースタグとして扱う形式。プレリリースやビルドメタデータは対象外。
	semverPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	// errNoTag は、起点となるリリースタグが 1 つも無いことを表す。
	errNoTag = xerrors.New("❌ リリースタグが存在しません。先に初期タグ(v0.0.0)を作成してください")
	// errNoTagForBranch は、ブランチ作成の起点となるタグが無いことを表す。
	errNoTagForBranch = xerrors.New("❌ 最新のリリースタグを取得できませんでした。初期タグ作成が必要です\n" +
		"➡️ 先に make tag-patch などで初期タグを作成してから再実行してください")
	// errUnknownBump は、-bump に未知の粒度が渡されたことを表す。
	errUnknownBump = xerrors.New("unknown -bump (patch / minor / major)")
	// errNoReleaseNote は、タグ本文にするリリースノートが production に無いことを表す。
	errNoReleaseNote = xerrors.New("が存在しません。タグとリリースをスキップしました")
	// errBranchExists は、作成しようとしたブランチが origin に既に在ることを表す。
	errBranchExists = xerrors.New("は既に存在します。処理を中止します")
	// errDirtyWorktree は、作業ツリーに未コミットの変更が残っていることを表す。
	errDirtyWorktree = xerrors.New("❌ 作業ツリーに未コミットの変更があります。変更をコミットまたは退避してから再実行してください")
)

// version は、リリースタグの意味的なバージョン。
type version struct {
	major, minor, patch int
}

// step は、実行する 1 コマンド。
type step struct {
	name string
	args []string
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < minArgs {
		log.Fatalf("❌ usage: release <tag|branch> [flags]")
	}

	var err error

	switch os.Args[1] {
	case "tag":
		err = runTag(os.Args[2:])
	case "branch":
		err = runBranch(os.Args[2:])
	default:
		log.Fatalf("❌ unknown subcommand (tag / branch)")
	}

	if err != nil {
		log.Fatalf("%v", err)
	}
}

// ---- バージョン解決（純粋） -------------------------------------------------

func (v version) String() string {
	return "v" + strconv.Itoa(v.major) + "." + strconv.Itoa(v.minor) + "." + strconv.Itoa(v.patch)
}

// parseVersion は、`vX.Y.Z` 形式の文字列を version へ変換します。
func parseVersion(s string) (version, bool) {
	m := semverPattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return version{}, false
	}

	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])

	return version{major: major, minor: minor, patch: patch}, true
}

// latestVersion は、`git tag` の出力から最新のリリースバージョンを選びます。
// 比較は数値ごとに行うため、v1.9.0 より v1.10.0 が新しいと判定されます。
func latestVersion(tagOutput string) (version, bool) {
	versions := make([]version, 0)

	for line := range strings.SplitSeq(tagOutput, "\n") {
		if v, ok := parseVersion(line); ok {
			versions = append(versions, v)
		}
	}

	if len(versions) == 0 {
		return version{}, false
	}

	sort.Slice(versions, func(i, j int) bool {
		a, b := versions[i], versions[j]
		if a.major != b.major {
			return a.major > b.major
		}

		if a.minor != b.minor {
			return a.minor > b.minor
		}

		return a.patch > b.patch
	})

	return versions[0], true
}

// bump は、指定の粒度で次のバージョンを返します。
func bump(v version, kind string) (version, error) {
	switch kind {
	case "patch":
		return version{major: v.major, minor: v.minor, patch: v.patch + 1}, nil
	case "minor":
		return version{major: v.major, minor: v.minor + 1}, nil
	case "major":
		return version{major: v.major + 1}, nil
	default:
		return version{}, xerrors.Wrap(errUnknownBump, kind)
	}
}

// ---- 手順の組み立て（純粋） -------------------------------------------------

func (s step) String() string { return s.name + " " + strings.Join(s.args, " ") }

// syncProductionSteps は、production を origin の最新へ合わせる手順を返します。
func syncProductionSteps() []step {
	return []step{
		{name: "git", args: []string{"fetch", "origin", "production"}},
		{name: "git", args: []string{"switch", "production"}},
		{name: "git", args: []string{"reset", "--hard", "origin/production"}},
		{name: "git", args: []string{"fetch", "--tags", "origin"}},
	}
}

// tagSteps は、リリースノートを本文にタグを打ち、GitHub Release を作る手順を返します。
func tagSteps(next, notePath string) []step {
	return []step{
		{name: "git", args: []string{"tag", "-a", next, "-F", notePath}},
		{name: "git", args: []string{"push", "origin", next}},
		{name: "gh", args: []string{"release", "create", next, "--title", next, "--notes-file", notePath}},
	}
}

// branchSteps は、base からブランチを切って push し、デフォルトブランチに設定する手順を返します。
func branchSteps(base, branch string) []step {
	return []step{
		{name: "git", args: []string{"fetch", "origin", base}},
		{name: "git", args: []string{"switch", "-c", branch, "origin/" + base}},
		{name: "git", args: []string{"push", "origin", branch}},
		{name: "gh", args: []string{"repo", "edit", "--default-branch", branch}},
	}
}

// notePath は、指定バージョンのリリースノートのパスを返します。
func notePath(v string) string { return releaseNoteDir + "/" + v + ".md" }

// ---- 実行 -------------------------------------------------------------------

// resolveNext は、最新タグを取得して次バージョン（現在, 次）を決めます。
func resolveNext(bumpKind string) (version, version, error) {
	if err := run(step{name: "git", args: []string{"fetch", "--tags", "origin"}}); err != nil {
		return version{}, version{}, err
	}

	out, err := output("git", "tag")
	if err != nil {
		return version{}, version{}, err
	}

	current, ok := latestVersion(out)
	if !ok {
		return version{}, version{}, errNoTag
	}

	next, err := bump(current, bumpKind)
	if err != nil {
		return version{}, version{}, err
	}

	return current, next, nil
}

func runTag(args []string) error {
	fs := flag.NewFlagSet("tag", flag.ExitOnError)
	bumpKind := fs.String("bump", "", "patch / minor / major")

	if err := fs.Parse(args); err != nil {
		return xerrors.Wrap(err, "failed to parse flags")
	}

	current, next, err := resolveNext(*bumpKind)
	if err != nil {
		return err
	}

	log.Printf("🔖 タグから最新タグバージョンを取得: %s", current)
	log.Printf("➡️ 次のリリースバージョンを作成: %s", next)

	// production を最新へ合わせてからリリースノートの有無を見る。順序は入れ替えないこと。
	// タグは production HEAD に打つので、確かめるべきは「production にノートがあるか」であり、
	// switch 前の作業ツリー（別ブランチ）にノートがあるかではない。
	log.Printf("🔄 productionブランチの最新を取得中...")

	if err := runAll(syncProductionSteps()); err != nil {
		return err
	}

	log.Printf("✅ 最新のproductionを取得完了")

	note := notePath(next.String())
	if _, err := os.Stat(note); err != nil {
		return xerrors.Wrap(errNoReleaseNote, "❌ "+note)
	}

	if err := runAll(tagSteps(next.String(), note)); err != nil {
		return err
	}

	log.Printf("✅ タグを打ちました %s on production HEAD", next)

	return nil
}

func runBranch(args []string) error {
	fs := flag.NewFlagSet("branch", flag.ExitOnError)
	bumpKind := fs.String("bump", "", "patch / minor / major")
	prefix := fs.String("prefix", "release", "ブランチ名の接頭辞 (release / hotfix)")
	base := fs.String("base", "production", "分岐元ブランチ")

	if err := fs.Parse(args); err != nil {
		return xerrors.Wrap(err, "failed to parse flags")
	}

	log.Printf("🔄 最新のタグを取得中...")

	current, next, err := resolveNext(*bumpKind)
	if err != nil {
		if xerrors.Is(err, errNoTag) {
			return errNoTagForBranch
		}

		return err
	}

	log.Printf("✅ 最新のタグを取得完了")

	branch := *prefix + "/" + next.String()

	log.Printf("🔖 タグから最新リリースバージョンを取得: 【 %s 】", current)
	log.Printf("➡️ 次のリリースバージョンを作成: 【 %s 】", next)
	log.Printf("🌱 ブランチを作成: %s → 【 %s 】", *base, branch)

	if remoteBranchExists(branch) {
		return xerrors.Wrap(errBranchExists, "❌ ブランチ【 "+branch+" 】")
	}

	status, err := output("git", "status", "--porcelain")
	if err != nil {
		return err
	}

	if strings.TrimSpace(status) != "" {
		_ = run(step{name: "git", args: []string{"status", "--short"}})

		return errDirtyWorktree
	}

	if err := runAll(branchSteps(*base, branch)); err != nil {
		return err
	}

	log.Printf("✅ デフォルトブランチを %s に切り替えて、プッシュしました。", branch)

	return nil
}

// remoteBranchExists は、origin に同名ブランチが既にあるかを返します。
func remoteBranchExists(branch string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	//nolint:gosec // branch はタグ由来のバージョン文字列
	return exec.CommandContext(ctx, "git", "ls-remote", "--exit-code", "--heads", "origin", branch).Run() == nil
}

func run(s step) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.name, s.args...) //nolint:gosec // 引数は本ファイル内で組み立てた固定手順
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return xerrors.Wrap(err, "failed: "+s.String())
	}

	return nil
}

func runAll(steps []step) error {
	for _, s := range steps {
		if err := run(s); err != nil {
			return err
		}
	}

	return nil
}

func output(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).Output() //nolint:gosec // 引数は本ファイル内の固定値
	if err != nil {
		return "", xerrors.Wrap(err, "failed: "+name+" "+strings.Join(args, " "))
	}

	return string(out), nil
}
