// audit パッケージは、操作ごとの監査イベントを構造化ログとして出力する（DESIGN §9.5）。
// イベントには時刻、操作、相対パス、判定、ルール名、結果、所要時間だけを含め、
// ファイル内容や秘密値は型として持たない。
package audit

import (
	"io"
	"log/slog"
	"os"
	"time"
)

// Decision の値。判定の前に操作が終わった場合は NotEvaluated を使う。
const (
	Allow        = "allow"
	Deny         = "deny"
	NotEvaluated = "not_evaluated"
)

// Event は 1 回の操作の記録。Path は検証済みの相対パスで、検証に失敗した入力では空にする。
// 生の入力をログに出さないことで、制御文字によるログ注入も防ぐ。
type Event struct {
	// Action は read、write、list、delete のいずれか。
	Action string
	// Path はルートからの正規化済みの相対パス。
	Path string
	// Decision は Allow、Deny、または NotEvaluated のいずれか。
	Decision string
	// Rule は判定を決めたルールの名前。既定拒否と未評価では空。
	Rule string
	// Result は ok、またはエラーコード。
	Result string
	// Duration は操作全体の所要時間。
	Duration time.Duration
}

// Logger は監査イベントを JSON Lines で書き出す。
type Logger struct {
	logger *slog.Logger
}

// New は w に JSON Lines で書き出す Logger を返す。
func New(w io.Writer) *Logger {
	return &Logger{logger: slog.New(slog.NewJSONHandler(w, nil))}
}

// NewStderr は標準エラー出力に書き出す Logger を返す。MCP の標準出力は使わない。
func NewStderr() *Logger {
	return New(os.Stderr)
}

// Log は e を 1 行の監査イベントとして書き出す。
func (l *Logger) Log(e Event) {
	l.logger.Info("operation",
		slog.String("action", e.Action),
		slog.String("path", e.Path),
		slog.String("decision", e.Decision),
		slog.String("rule", e.Rule),
		slog.String("result", e.Result),
		slog.Float64("duration_ms", float64(e.Duration)/float64(time.Millisecond)),
	)
}
