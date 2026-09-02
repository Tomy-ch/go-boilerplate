package architest

// 集約境界の隔離規約（internal/domain/README.md の Cross-aggregate reference）を機械的に検査する。
//
// domain パッケージが他の domain パッケージを import してよいのは次の 2 つだけ:
//
//  1. internal/domain/lexicon 配下（集約横断で共有する業務 VO。ADR-0039）
//  2. 自分の集約の internal/ 配下（sub-entity。他集約からは Go 自身が到達を塞ぐ）
//
// この規約が lint ではなくここに居るのは、depguard が importer を files でしか見られず
// allow / deny は imported しか見ないためである。「自分の集約の internal/ 配下だけ許す」は
// 2 端点の関係なので、depguard の 1 本のルールでは原理的に書けない。
//
// import は正規表現で拾う（realtime_isolation_test.go と同じ方式）。build tag で除外される
// ファイルの import も拾うため過大近似になるが、禁止規則としては安全側に倒れる。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// domainRoot は、集約が並ぶディレクトリ（repo root 相対）。
const domainRoot = "internal/domain"

// domainImportPrefix は、domain パッケージを指す import path の接頭辞。
const domainImportPrefix = "go-boilerplate/internal/domain/"

// lexiconAggregate は、集約横断で共有する業務 VO の置き場。どの集約からも import してよい。
const lexiconAggregate = "lexicon"

// serviceAggregate は、Domain Service の置き場。集約の import を許されており本規約の対象外
// （.golangci-full.yaml の maintain_a_sound_domain_service が受け持つ）。
const serviceAggregate = "service"

// subEntityMarker は、sub-entity のパッケージを他集約から隔離する目印。Go 自身が
// internal/ の外からの import を拒否するため、この規約と二重に守られる。
const subEntityMarker = "internal"

func TestDomainAggregateImportIsolation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("どの集約も他集約を直接 import していない", func(t *testing.T) {
			t.Parallel()

			violations, err := collectDomainAggregateViolations(moduleRoot(t))
			require.NoError(t, err)
			assert.Empty(t, violations,
				"domain の集約が他集約を直接 import している。集約跨ぎの参照は識別子だけにし、"+
					"共有したい業務 VO は internal/domain/lexicon へ置く"+
					"（internal/domain/README.md の Cross-aggregate reference）")
		})

		t.Run("走査対象の集約が実在する", func(t *testing.T) {
			t.Parallel()

			// 走査が空になっても Empty は通ってしまうため、対象が実在することを別に主張して
			// 検査の縮退をここで止める。
			seen, err := collectScannedDomainAggregates(moduleRoot(t))
			require.NoError(t, err)
			assert.Contains(t, seen, "cart", "internal/domain/cart が走査対象に入っていない")
			assert.Contains(t, seen, "user", "internal/domain/user が走査対象に入っていない")
		})
	})
}

func Test_domainImportViolation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lexicon はどの集約からも許す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, domainImportViolation("cart", domainImportPrefix+"lexicon/money"))
		})

		t.Run("自分の集約の internal 配下は許す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, domainImportViolation("inquiry", domainImportPrefix+"inquiry/internal/message"))
		})

		t.Run("domain 以外の import は対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, domainImportViolation("cart", "go-boilerplate/pkg/uuid"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("他集約の直接 import は違反", func(t *testing.T) {
			t.Parallel()
			assert.True(t, domainImportViolation("cart", domainImportPrefix+"inquiry"))
		})

		t.Run("他集約の internal 配下も違反", func(t *testing.T) {
			t.Parallel()
			// Go 自身も塞ぐが、規約としてもここで落とす。
			assert.True(t, domainImportViolation("cart", domainImportPrefix+"inquiry/internal/message"))
		})

		t.Run("自分の配下でも internal を経由しないネストは違反", func(t *testing.T) {
			t.Parallel()
			// product/category は自前の Repository を持つ別集約であり sub-entity ではない。
			// パス形状だけでは sub-entity と区別できないため、internal/ を目印にして分ける。
			assert.True(t, domainImportViolation("product", domainImportPrefix+"product/category"))
		})

		t.Run("sub-entity から親の Root への import は違反", func(t *testing.T) {
			t.Parallel()
			// sub-entity は親への逆参照を持たない（internal/domain/README.md）。
			assert.True(t, domainImportViolation("inquiry", domainImportPrefix+"inquiry"))
		})
	})
}

func Test_aggregateOfDomainPath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集約直下のファイルから集約名を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "cart", aggregateOfDomainPath("internal/domain/cart/cart_domain.go"))
		})

		t.Run("ネストしたファイルでも最上位の集約名を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "inquiry", aggregateOfDomainPath("internal/domain/inquiry/internal/message/message.go"))
		})

		t.Run("domain 配下でなければ空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, aggregateOfDomainPath("internal/usecase/inquiry/inquiry_usecase.go"))
		})
	})
}

// collectDomainAggregateViolations は、domain 配下の production code が持つ規約違反の import を
// `file -> import` の形で返す。
func collectDomainAggregateViolations(root string) ([]string, error) {
	var violations []string

	err := walkProductionGoFiles(filepath.Join(root, domainRoot), func(path string, content []byte) {
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)

		aggregate := aggregateOfDomainPath(rel)
		if aggregate == "" || aggregate == serviceAggregate {
			return
		}

		for _, m := range internalImportRe.FindAllStringSubmatch(string(content), -1) {
			if domainImportViolation(aggregate, m[1]) {
				violations = append(violations, rel+" -> "+m[1])
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return violations, nil
}

// collectScannedDomainAggregates は、走査が実際に触れた集約名を返す。
func collectScannedDomainAggregates(root string) ([]string, error) {
	seen := map[string]struct{}{}

	err := walkProductionGoFiles(filepath.Join(root, domainRoot), func(path string, _ []byte) {
		rel, _ := filepath.Rel(root, path)
		if aggregate := aggregateOfDomainPath(filepath.ToSlash(rel)); aggregate != "" {
			seen[aggregate] = struct{}{}
		}
	})
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}

	return out, nil
}

// domainImportViolation は、aggregate に属するファイルからの imp が規約違反かを返す。
func domainImportViolation(aggregate, imp string) bool {
	rest, ok := strings.CutPrefix(imp, domainImportPrefix)
	if !ok {
		return false
	}

	target, sub, _ := strings.Cut(rest, "/")
	switch {
	case target == lexiconAggregate:
		return false
	case target != aggregate:
		return true
	default:
		// 自分の集約の中でも、sub-entity の目印を経由しないネストは別集約とみなす。
		return sub != subEntityMarker && !strings.HasPrefix(sub, subEntityMarker+"/")
	}
}

// aggregateOfDomainPath は、repo root 相対のパスから集約名を返す。domain 配下でなければ空を返す。
func aggregateOfDomainPath(rel string) string {
	rest, ok := strings.CutPrefix(rel, domainRoot+"/")
	if !ok {
		return ""
	}

	aggregate, _, found := strings.Cut(rest, "/")
	if !found {
		return ""
	}

	return aggregate
}
