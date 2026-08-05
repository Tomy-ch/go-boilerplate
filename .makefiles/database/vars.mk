# 対象DB（local / test / prd など）。未指定なら所有している local 系データベース
# （主 checkout なら local、スロット取得済み worktree なら wt<N>_local）。
# 所有権の不変条件と DB_LOCAL の定義は .makefiles/database/pool.mk を参照。
DB ?= $(DB_LOCAL)
# 作業ディレクトリ
work-dir ?= /app
# マージDMLタイプ
type ?=
