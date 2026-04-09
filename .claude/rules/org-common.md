# CloudPanda-Inc 共通ルール

> このファイルは [workflows リポ](https://github.com/CloudPanda-Inc/workflows) の `templates/CLAUDE-common.md` から
> 各リポの `.claude/rules/org-common.md` へ自動配信される。
> 編集する場合は workflows リポ側を変更すること。手動で直接編集しても次回の同期で上書きされる。

## 基本思想: 第一原理とエンジニアリングの5ステップ

既存のやり方や前例を鵜呑みにせず、「そもそも何が本当に必要か」を第一原理から考えること。

1. **要件を疑え** — そもそもその要件は正しいか？バカな要件を最適化しても意味がない
2. **削除しろ** — 不要な部品・工程を消せ。消した後で必要だとわかったら戻せばいい
3. **簡素化しろ** — 残ったものを単純にしろ。削除の前に簡素化するな
4. **サイクルを速くしろ** — 速く回せ。ただし1-3の前に速くするな
5. **自動化しろ** — 自動化は最後。1-4で磨いたプロセスだけを自動化する

Issue 作成・改善提案・自動実装など、あらゆる作業でこの順番を守ること。

## 基本原則

**だれにでもわかりやすく。**

設計、命名、コード、コメント、PR・Issue の文章、ドキュメント、すべてにおいて「初見の人が読んで迷わない」を最優先にする。簡潔さと明快さが衝突したら、明快さを取る。

## 設計原則: YAGNI

シンプルで堅牢な設計を優先する。機能を盛るより、障害点を減らす。動く部品が少ないほど壊れにくく、保守しやすい。「将来起きるかもしれない」問題への過剰な予防策は、今ある問題よりも有害になりうる。

- **今必要なものだけ作る**: 「将来必要になるかもしれない」は作らない理由として十分
- **既存の仕組みで済むなら新しい仕組みを作らない**: 新しいワークフロー、lint ルール、監視を追加する前に、既存のもので対応できないか検討する
- **1つのバグから umbrella Issue を展開しない**: 具体的なバグには具体的な修正。同パターン課題の探索（同パターンの grep 一括修正）は推奨だが、「メタ改善」として子 Issue を量産しない
- **「念のため」は理由にならない**: 実害・実益を具体的に説明できない提案は不要

## 言語

コメント、テストの説明文（`describe` / `it` / `context`）、ログメッセージは日本語で書く。変数名・メソッド名・クラス名は英語。

## Git

- コミットメッセージは `feat:` `fix:` `chore:` `cleanup:` `sync:` 等の接頭辞 + 命令形で 72 文字以内
- push 前に lint とテストを実行する
- 秘密情報（API キー、パスワード、トークン、`.env` ファイル）をコミットしない
- セッション関連ファイル（`cookies.txt` 等）をコミットしない

## GitHub ラベル

ラベル名は日本語で統一する（英語の固有名詞・略語は許容）。新しいラベルの作成前に既存ラベルを確認すること。`bug`、`enhancement` 等の英語ラベルは使わない。

## CI ランナー

| 条件 | ランナー | 理由 |
|------|---------|------|
| Ruby + `bundle install` が必要 | `CI_RUNNER_LABEL`（Mac Studio） | tool cache があり高速 |
| Ruby 不要（Go, Python, シェル, GitHub API のみ） | `ubicloud-standard-2` | Mac Studio の枠を消費しない |
| Docker が必要 | `ubicloud-standard-2` | Mac Studio は Docker 未対応 |

全ワークフローで `runs-on` にバーストルーティング式を使用する:

```yaml
runs-on: ${{ (vars.CI_BURST_ACTIVE != '' && github.run_attempt > 1) && 'ubicloud-standard-2' || vars.CI_RUNNER_LABEL || 'ubuntu-latest' }}
```

## Issue・PR 作成前の検証

Issue や PR を作る前に、以下を実行すること。事実誤認の Issue は読む側の時間を奪い、信頼を損なう。

1. **既存の Issue・PR を検索して重複がないか確認する**
2. **実装コードを読む** — PR 本文や記憶の記述を鵜呑みにしない
3. **本当に問題があるか検証する**
4. **本当に必要か判断する** — 実害の大きさ、メンテナンスコストに見合うか
5. **改善案は対処療法・根本療法・メタ改善の 3 層で整理する**
6. **メタ改善の 9 軸を常に検討する** — 同パターン課題・自動化・予防・監視・仕組み化・未来想定・フィードバックループ・劣化検知・知識化

## `.claude/settings.json`

各リポに `.claude/settings.json` を配置し、以下の保護を有効にする:

- **Allow**: 読み取り系ツール（Read, Glob, Grep）、安全な git コマンド（status, diff, log, show, branch, fetch）、リポ固有の安全コマンド
- **Deny**: 破壊的 git 操作（force push, hard reset, clean, checkout --）、機密ファイルの読み取り（`.env`, credential files）、`rm -rf /*`

## この共通ルールの運用

- **Single Source of Truth**: [`workflows/templates/CLAUDE-common.md`](https://github.com/CloudPanda-Inc/workflows/blob/main/templates/CLAUDE-common.md)
- **配信**: `sync-org-standards.yml` が main push 時に全リポの `.claude/rules/org-common.md` へ自動 PR を作成
- **各リポの `CLAUDE.md`**: リポ固有のルール（コマンド、テスト方針、プロジェクト構造、ドメインガイダンス）のみ記載
- **共通ルールの変更**: workflows リポで `templates/CLAUDE-common.md` を編集 → マージ → 全リポに自動配信
- **リポ固有の上書き**: `.claude/rules/org-common.md` より `CLAUDE.md` が優先される。共通ルールと異なる運用が必要な場合はリポの `CLAUDE.md` に明示的に記載する
