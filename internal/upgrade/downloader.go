// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0

package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
)

// DownloadConfig holds download configuration.
type DownloadConfig struct {
	MaxRetries int
	// MaxBytes 非零时同时限制响应声明大小和实际下载量；零保持既有行为。
	MaxBytes         int64
	CheckRedirect    func(*http.Request, []*http.Request) error
	Timeout          time.Duration
	IdleTimeout      time.Duration
	EventCallback    func(DownloadEvent)
	ProgressCallback func(downloaded, total int64)
	ProgressInterval time.Duration
}

// DownloadEvent 报告实际请求、重试原因和已接收字节，供交互式升级显示。
type DownloadEvent struct {
	URL                  string
	Attempt, MaxAttempts int
	Downloaded           int64
	Err                  error
}

var downloadAfter = time.After

type downloadFile interface {
	io.Writer
	io.Closer
	Stat() (os.FileInfo, error)
}

var downloadOpenFile = func(name string, flags int, mode os.FileMode) (downloadFile, error) {
	return os.OpenFile(name, flags, mode)
}

// DefaultDownloadConfig returns sensible download defaults.
func DefaultDownloadConfig() *DownloadConfig {
	return &DownloadConfig{
		MaxRetries:       3,
		Timeout:          10 * time.Minute,
		IdleTimeout:      30 * time.Second,
		ProgressInterval: 200 * time.Millisecond,
	}
}

// Download fetches url to destPath with default config.
func Download(url, destPath string) (int64, error) {
	return DownloadWithConfig(context.Background(), url, destPath, DefaultDownloadConfig())
}

// DownloadWithProgress downloads a file and reports progress.
func DownloadWithProgress(ctx context.Context, url, destPath string, showProgress func(percent float64, downloaded, total int64)) (int64, error) {
	cfg := DefaultDownloadConfig()
	cfg.ProgressCallback = func(downloaded, total int64) {
		if showProgress != nil && total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			showProgress(percent, downloaded, total)
		}
	}
	return DownloadWithConfig(ctx, url, destPath, cfg)
}

// DownloadWithConfig fetches a file with custom configuration and retries.
func DownloadWithConfig(ctx context.Context, url, destPath string, cfg *DownloadConfig) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return 0, fmt.Errorf("创建目录失败: %w", err)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries < 1 {
		maxRetries = 1
	}

	var lastErr error
	state := downloadResume{}
	defer func() {
		if lastErr != nil {
			_ = os.Remove(destPath)
		}
	}()
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Second * time.Duration(1<<uint(attempt-1))
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-downloadAfter(backoff):
			}
		}

		if cfg.EventCallback != nil {
			cfg.EventCallback(DownloadEvent{URL: url, Attempt: attempt + 1, MaxAttempts: maxRetries, Downloaded: state.offset, Err: lastErr})
		}
		n, err := doDownloadAttempt(ctx, url, destPath, cfg, &state)
		if err == nil {
			lastErr = nil
			return n, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if !isRetriable(err) {
			return 0, err
		}
		if state.validator == "" || state.total <= 0 {
			_ = os.Remove(destPath)
			state = downloadResume{}
		}
	}

	return 0, fmt.Errorf("下载失败 (尝试 %d 次后): %w", maxRetries, lastErr)
}

type downloadResume struct {
	offset, total int64
	validator     string
}

type downloadPolicyError struct{ error }

func (e downloadPolicyError) Unwrap() error { return e.error }

func doDownload(ctx context.Context, url, destPath string, cfg *DownloadConfig) (int64, error) {
	return doDownloadAttempt(ctx, url, destPath, cfg, &downloadResume{})
}

func downloadNetworkError(err error) error {
	reason, message := "download_network_error", "下载网络请求失败"
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		reason, message = "download_timeout", "下载超时，请检查网络或代理后重试"
	}
	return apperrors.NewAPI(message+": "+err.Error(), apperrors.WithReason(reason), apperrors.WithRetryable(isRetriable(err)), apperrors.WithCause(err), apperrors.WithHint("检查网络或代理配置后重新执行 dws upgrade"))
}

func doDownloadAttempt(ctx context.Context, url, destPath string, cfg *DownloadConfig, state *downloadResume) (int64, error) {
	requestCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	var timer *time.Timer
	if cfg.IdleTimeout > 0 {
		timer = time.AfterFunc(cfg.IdleTimeout, func() { cancel(context.DeadlineExceeded) })
		defer timer.Stop()
	}
	req, err := http.NewRequestWithContext(requestCtx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	// 禁止传输层解压，以保证 Range、Content-Length 和校验的字节语义一致。
	req.Header.Set("Accept-Encoding", "identity")
	if state.offset > 0 && state.validator != "" {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", state.offset))
		req.Header.Set("If-Range", state.validator)
	}
	client := &http.Client{CheckRedirect: cfg.CheckRedirect, Timeout: cfg.Timeout}
	if cfg.CheckRedirect != nil {
		client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
			if err := cfg.CheckRedirect(next, via); err != nil {
				return downloadPolicyError{err}
			}
			return nil
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		var policy downloadPolicyError
		if errors.As(err, &policy) {
			return 0, err
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if cause := context.Cause(requestCtx); cause != nil {
			err = cause
		}
		return 0, downloadNetworkError(err)
	}
	defer func() { _ = resp.Body.Close() }()
	offset := int64(0)
	total := resp.ContentLength
	validator := resp.Header.Get("ETag")
	if strings.HasPrefix(validator, "W/") {
		validator = ""
	}
	if validator == "" {
		validator = resp.Header.Get("Last-Modified")
	}
	switch resp.StatusCode {
	case http.StatusOK:
		// 源忽略 Range 或 If-Range 不匹配时覆盖重下，绝不追加不同版本的内容。
	case http.StatusPartialContent:
		var start, end, whole int64
		header := resp.Header.Get("Content-Range")
		count, scanErr := fmt.Sscanf(header, "bytes %d-%d/%d", &start, &end, &whole)
		if scanErr != nil || count != 3 || header != fmt.Sprintf("bytes %d-%d/%d", start, end, whole) || state.offset <= 0 || start != state.offset || end < start || end != whole-1 || whole != state.total || resp.ContentLength != end-start+1 || validator != state.validator {
			return 0, fmt.Errorf("下载续传响应与原文件不一致")
		}
		offset, total = start, whole
	default:
		return 0, apperrors.NewAPI(fmt.Sprintf("下载失败 (HTTP %d)", resp.StatusCode), apperrors.WithReason("download_http_status"), apperrors.WithRetryable(resp.StatusCode == 429 || resp.StatusCode >= 500), apperrors.WithDetails(map[string]any{"status": resp.StatusCode}))
	}
	if cfg.MaxBytes > 0 && total > cfg.MaxBytes {
		return 0, fmt.Errorf("下载文件超过大小上限 %d 字节", cfg.MaxBytes)
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if offset > 0 {
		flags = os.O_WRONLY | os.O_APPEND
	}
	out, err := downloadOpenFile(destPath, flags, 0600)
	if err != nil {
		return 0, fmt.Errorf("创建文件失败: %w", err)
	}
	defer func() { _ = out.Close() }()
	if offset > 0 {
		info, statErr := out.Stat()
		if statErr != nil {
			return 0, statErr
		}
		if info.Size() != offset {
			return 0, fmt.Errorf("下载续传文件大小已变化")
		}
	}
	state.offset, state.total, state.validator = offset, total, validator
	var writer io.Writer = downloadDiskWriter{out}
	if cfg.ProgressCallback != nil {
		pw := &progressWriter{writer: writer, total: total, downloaded: offset, callback: cfg.ProgressCallback, interval: cfg.ProgressInterval, lastCallback: time.Now()}
		defer pw.finalProgress()
		writer = pw
	}
	var body io.Reader = downloadBodyReader{reader: resp.Body, ctx: ctx, requestCtx: requestCtx, timer: timer, idle: cfg.IdleTimeout}
	if cfg.MaxBytes > 0 {
		body = io.LimitReader(body, cfg.MaxBytes-offset+1)
	}
	n, err := io.Copy(writer, body)
	state.offset += n
	if err != nil {
		return 0, err
	}
	if cfg.MaxBytes > 0 && state.offset > cfg.MaxBytes {
		return 0, fmt.Errorf("下载文件超过大小上限 %d 字节", cfg.MaxBytes)
	}
	if err := out.Close(); err != nil {
		return 0, fmt.Errorf("关闭下载文件失败: %w", err)
	}
	return state.offset, nil
}

type downloadDiskWriter struct{ io.Writer }

func (w downloadDiskWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err != nil {
		err = fmt.Errorf("写入下载文件失败: %w", err)
	}
	return n, err
}

type downloadBodyReader struct {
	reader          io.Reader
	ctx, requestCtx context.Context
	timer           *time.Timer
	idle            time.Duration
}

func (r downloadBodyReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.timer != nil {
		r.timer.Reset(r.idle)
	}
	if err != nil && err != io.EOF {
		if r.ctx.Err() != nil {
			return n, r.ctx.Err()
		}
		if cause := context.Cause(r.requestCtx); cause != nil {
			err = cause
		}
		err = downloadNetworkError(err)
	}
	return n, err
}

// progressWriter wraps an io.Writer and reports download progress.
type progressWriter struct {
	writer       io.Writer
	total        int64
	downloaded   int64
	callback     func(downloaded, total int64)
	interval     time.Duration
	lastCallback time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.downloaded += int64(n)

	now := time.Now()
	if pw.callback != nil && now.Sub(pw.lastCallback) >= pw.interval {
		pw.callback(pw.downloaded, pw.total)
		pw.lastCallback = now
	}
	return n, err
}

func (pw *progressWriter) finalProgress() {
	if pw.callback != nil {
		pw.callback(pw.downloaded, pw.total)
	}
}

func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	var typed *apperrors.Error
	if errors.As(err, &typed) && typed.RetryableSet {
		return typed.Retryable
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return true
	}
	errStr := err.Error()
	for _, pattern := range []string{"connection reset", "connection refused", "timeout", "temporary failure", "eof"} {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	return false
}
