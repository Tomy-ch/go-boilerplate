// Package dbpool は、複数の git worktree が単一共有 Postgres を per-worktree のデータベース
// （wt<N>_local / wt<N>_test）として貸し借りする DB スロットプールのコアロジックを提供します。
// リース（ロックディレクトリ）管理・stale 回収・DB 作成・compose 起動を担い、cmd/db-pool.go から
// 実依存を配線して呼び出します。詳細は docs/maintenance/db-worktree-pool.md。
package dbpool

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	// dirPerm は、レジストリ / ロックディレクトリのパーミッション（ユーザー専有）。
	dirPerm = 0o700
	// filePerm は、meta / lock ファイルのパーミッション（ユーザー専有）。
	filePerm = 0o600
)

// reclaimSeq は、回収時の一時名を一意にするためのカウンタ（同一プロセス内の goroutine 間衝突回避）。
var reclaimSeq atomic.Int64

// Meta は、スロットのリース保持者情報です。
type Meta struct {
	Owner     string // リース保持 worktree の絶対パス
	Heartbeat time.Time
	Branch    string
	Slot      int
}

// Registry は、ホスト上のロックディレクトリでスロットのリースを管理します。
// os.Mkdir の原子性で新規取得と stale 回収の双方を排他し、二重リースを防ぎます。
type Registry struct {
	dir      string           // リースレジストリのルート（例: ~/.cache/gobp-db-pool）
	owner    string           // 自 worktree の絶対パス
	branch   string           // 自 worktree の現在ブランチ
	ttl      time.Duration    // heartbeat 失効までの猶予
	maxSlots int              // スロット数
	now      func() time.Time // テスト用に時刻を差し替え可能にする
}

// NewRegistry は Registry を生成します。now が nil の場合は time.Now を使います。
func NewRegistry(dir, owner, branch string, ttl time.Duration, maxSlots int, now func() time.Time) *Registry {
	if now == nil {
		now = time.Now
	}
	return &Registry{dir: dir, owner: owner, branch: branch, ttl: ttl, maxSlots: maxSlots, now: now}
}

// Lock は、スロット走査中の取得（fresh / reclaim）を全プロセス・全 goroutine 横断で直列化する
// アドバイザリロック（flock）を取得します。fresh mkdir と reclaim rename の間の隙間で二重取得が起きる
// のを根本的に防ぎます。戻り値の関数で解放します。EnsureDir 後に呼ぶこと。
func (r *Registry) Lock() (func(), error) {
	f, err := os.OpenFile(filepath.Join(r.dir, ".scan.lock"), os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to open scan lock")
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, xerrors.Wrap(err, "failed to acquire scan lock")
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// EnsureDir は、レジストリを用意します。symlink は先読み攻撃対策として拒否し、0700 に制限します。
func (r *Registry) EnsureDir() error {
	if fi, err := os.Lstat(r.dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return xerrors.New(fmt.Sprintf("pool dir %q is a symlink; set GOBP_DB_POOL_DIR to a safe path", r.dir))
	}
	if err := os.MkdirAll(r.dir, dirPerm); err != nil {
		return xerrors.Wrap(err, "failed to create pool dir")
	}
	_ = os.Chmod(r.dir, dirPerm)
	return nil
}

// MaxSlots は、スロット数を返します。
func (r *Registry) MaxSlots() int { return r.maxSlots }

// TryAcquireFresh は、未使用スロットを os.Mkdir の原子性で取得します（成功で true）。
func (r *Registry) TryAcquireFresh(slot int) bool {
	return os.Mkdir(r.lockDir(slot), dirPerm) == nil
}

// TryReclaim は、stale なスロットを原子的に回収します。既存ロックディレクトリを一意名へ rename して
// 「占有権」を原子的に奪い（rename の source は一度しか成功しないため、同時に狙う別プロセス/goroutine の
// うち勝者は1つだけ）、その後に作り直します。rename に失敗＝先取りされた場合は false を返します。
// 単純な rm→mkdir は各実行が互いの mkdir を消して全員成功しうるため使わない。
func (r *Registry) TryReclaim(slot int) bool {
	d := r.lockDir(slot)
	tmp := fmt.Sprintf("%s.claim.%d.%d", d, os.Getpid(), reclaimSeq.Add(1))
	if os.Rename(d, tmp) != nil {
		return false // 別プロセス/goroutine が先に回収した（source が既に消えた）
	}
	_ = os.RemoveAll(tmp)
	return os.Mkdir(d, dirPerm) == nil
}

// Exists は、スロットのロックが存在するかを返します。
func (r *Registry) Exists(slot int) bool {
	fi, err := os.Stat(r.lockDir(slot))
	return err == nil && fi.IsDir()
}

// WriteMeta は、スロットの meta を 0600 で書き出します。
func (r *Registry) WriteMeta(slot int) error {
	m := fmt.Sprintf("owner=%s\nheartbeat=%d\nbranch=%s\nslot=%d\n",
		r.owner, r.now().Unix(), r.branch, slot)
	if err := os.WriteFile(r.metaPath(slot), []byte(m), filePerm); err != nil {
		return xerrors.Wrap(err, "failed to write slot meta")
	}
	return nil
}

// ReadMeta は、スロットの meta を読み取ります（存在しなければ ok=false）。
func (r *Registry) ReadMeta(slot int) (Meta, bool) {
	f, err := os.Open(r.metaPath(slot))
	if err != nil {
		return Meta{}, false
	}
	defer func() { _ = f.Close() }()

	m := Meta{Slot: slot}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		switch k {
		case "owner":
			m.Owner = v
		case "branch":
			m.Branch = v
		case "heartbeat":
			if sec, err := strconv.ParseInt(v, 10, 64); err == nil {
				m.Heartbeat = time.Unix(sec, 0)
			}
		}
	}
	if sc.Err() != nil {
		return Meta{}, false
	}
	return m, true
}

// OwnedBySelf は、スロットが自 worktree の保持かを返します。
func (r *Registry) OwnedBySelf(slot int) bool {
	m, ok := r.ReadMeta(slot)
	return ok && m.Owner == r.owner
}

// IsStale は、他 worktree のリースが heartbeat TTL を超過しているかを返します。
// 自分の保持スロットは stale 扱いにしません（誤って自 DB を回収しないため）。
func (r *Registry) IsStale(slot int) bool {
	m, ok := r.ReadMeta(slot)
	if !ok {
		return true
	}
	if m.Owner == r.owner {
		return false
	}
	if m.Heartbeat.IsZero() {
		return true
	}
	return r.now().Sub(m.Heartbeat) > r.ttl
}

// Release は、スロットのロックを解放します（自分の保持のときのみ）。
func (r *Registry) Release(slot int) {
	if r.OwnedBySelf(slot) {
		_ = os.RemoveAll(r.lockDir(slot))
	}
}

// AgeSeconds は、スロットの heartbeat からの経過秒数を返します（meta 無しは -1）。
func (r *Registry) AgeSeconds(slot int) int64 {
	m, ok := r.ReadMeta(slot)
	if !ok || m.Heartbeat.IsZero() {
		return -1
	}
	return int64(r.now().Sub(m.Heartbeat).Seconds())
}

func (r *Registry) lockDir(slot int) string {
	return filepath.Join(r.dir, fmt.Sprintf("slot-%d.lock", slot))
}

func (r *Registry) metaPath(slot int) string {
	return filepath.Join(r.lockDir(slot), "meta")
}
