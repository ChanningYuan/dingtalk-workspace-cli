// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0

package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/testseam"
)

func TestCrossPlatformCoverageDownloadBodyTimeoutClassification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "x")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	cfg := DefaultDownloadConfig()
	cfg.MaxRetries = 1
	cfg.Timeout = 40 * time.Millisecond
	_, err := DownloadWithConfig(t.Context(), srv.URL, filepath.Join(t.TempDir(), "out"), cfg)
	var typed *apperrors.Error
	if !errors.As(err, &typed) || typed.Category != apperrors.CategoryAPI || typed.Reason != "download_timeout" || !typed.Retryable || !errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "写入文件失败") {
		t.Fatalf("网络读取超时没有正确分类: %v", err)
	}
}

func TestCrossPlatformCoverageDownloadResumesValidatedPartialBody(t *testing.T) {
	ready := make(chan time.Time)
	close(ready)
	testseam.Swap(t, &downloadAfter, func(time.Duration) <-chan time.Time { return ready })
	for _, ignoreRange := range []bool{false, true} {
		t.Run(fmt.Sprint(ignoreRange), func(t *testing.T) {
			var attempts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", `"fixture-v1"`)
				if attempts.Add(1) == 1 {
					w.Header().Set("Content-Length", "10")
					_, _ = io.WriteString(w, "0123")
					return
				}
				if r.Header.Get("Range") != "bytes=4-" || r.Header.Get("If-Range") != `"fixture-v1"` {
					t.Errorf("未请求带版本校验的续传: %v", r.Header)
				}
				if ignoreRange {
					_, _ = io.WriteString(w, "0123456789")
					return
				}
				w.Header().Set("Content-Range", "bytes 4-9/10")
				w.Header().Set("Content-Length", "6")
				w.WriteHeader(http.StatusPartialContent)
				_, _ = io.WriteString(w, "456789")
			}))
			defer srv.Close()
			dest := filepath.Join(t.TempDir(), "out")
			n, err := DownloadWithConfig(t.Context(), srv.URL, dest, DefaultDownloadConfig())
			body, readErr := os.ReadFile(dest)
			if err != nil || readErr != nil || n != 10 || string(body) != "0123456789" || attempts.Load() != 2 {
				t.Fatalf("续传损坏内容: n=%d body=%q err=%v read=%v", n, body, err, readErr)
			}
		})
	}
}

func TestCrossPlatformCoverageDownloadIdleTimeoutAndActiveStream(t *testing.T) {
	for _, stall := range []bool{true, false} {
		t.Run(fmt.Sprint(stall), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "8")
				for i := 0; i < 8; i++ {
					_, _ = io.WriteString(w, "x")
					w.(http.Flusher).Flush()
					if stall {
						<-r.Context().Done()
						return
					}
					time.Sleep(10 * time.Millisecond)
				}
			}))
			defer srv.Close()
			cfg := DefaultDownloadConfig()
			cfg.MaxRetries = 1
			cfg.IdleTimeout = 50 * time.Millisecond
			n, err := DownloadWithConfig(t.Context(), srv.URL, filepath.Join(t.TempDir(), "out"), cfg)
			if stall && (err == nil || !errors.Is(err, context.DeadlineExceeded)) {
				t.Fatalf("未中止停滞下载: %v", err)
			}
			if !stall && (err != nil || n != 8) {
				t.Fatalf("持续传输不应触发空闲超时: n=%d err=%v", n, err)
			}
		})
	}
}

func TestCrossPlatformCoverageDownloadRejectsUnsafeRecovery(t *testing.T) {
	ready := make(chan time.Time)
	close(ready)
	testseam.Swap(t, &downloadAfter, func(time.Duration) <-chan time.Time { return ready })
	for _, scenario := range []string{"range-start", "range-version", "range-total", "range-length", "missing-validator", "redirect-policy", "size-limit"} {
		t.Run(scenario, func(t *testing.T) {
			var attempts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if scenario == "redirect-policy" {
					http.Redirect(w, r, "/forbidden", http.StatusFound)
					return
				}
				if scenario == "size-limit" {
					_, _ = io.WriteString(w, "oversized")
					return
				}
				if scenario != "missing-validator" {
					w.Header().Set("ETag", `"v1"`)
				}
				if attempts.Add(1) == 1 {
					w.Header().Set("Content-Length", "10")
					_, _ = io.WriteString(w, "0123")
					return
				}
				if scenario == "missing-validator" {
					if r.Header.Get("Range") != "" {
						t.Error("没有版本标记时不允许续传")
					}
					_, _ = io.WriteString(w, "0123456789")
					return
				}
				w.Header().Set("Content-Range", "bytes 4-9/10")
				w.Header().Set("Content-Length", "6")
				switch scenario {
				case "range-start":
					w.Header().Set("Content-Range", "bytes 3-9/10")
				case "range-version":
					w.Header().Set("ETag", `"v2"`)
				case "range-total":
					w.Header().Set("Content-Range", "bytes 4-10/11")
				case "range-length":
					w.Header().Set("Content-Length", "5")
				}
				w.WriteHeader(http.StatusPartialContent)
				_, _ = io.WriteString(w, "456789")
			}))
			defer srv.Close()
			cfg := DefaultDownloadConfig()
			if scenario == "redirect-policy" {
				cfg.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("禁止此重定向") }
			}
			if scenario == "size-limit" {
				cfg.MaxBytes = 2
			}
			dest := filepath.Join(t.TempDir(), "out")
			_, err := DownloadWithConfig(t.Context(), srv.URL, dest, cfg)
			if scenario == "missing-validator" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || downloadCanFallback(err) {
				t.Fatalf("不安全的恢复允许换源: %v", err)
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatalf("失败的部分下载未清理: %v", err)
			}
		})
	}
}

type failingDownloadFile struct {
	*os.File
	failure string
}

func (f failingDownloadFile) Write(p []byte) (int, error) {
	if f.failure == "write" {
		return 0, os.ErrPermission
	}
	return f.File.Write(p)
}
func (f failingDownloadFile) Close() error {
	err := f.File.Close()
	if f.failure == "close" {
		return os.ErrPermission
	}
	return err
}
func (f failingDownloadFile) Stat() (os.FileInfo, error) {
	if f.failure == "stat" {
		return nil, os.ErrPermission
	}
	info, err := f.File.Stat()
	if f.failure == "size" && err == nil {
		return downloadSizeInfo{FileInfo: info}, nil
	}
	return info, err
}

type downloadSizeInfo struct{ os.FileInfo }

func (downloadSizeInfo) Size() int64 { return 0 }

func TestCrossPlatformCoverageDownloadFileErrorsStayLocal(t *testing.T) {
	ready := make(chan time.Time)
	close(ready)
	testseam.Swap(t, &downloadAfter, func(time.Duration) <-chan time.Time { return ready })
	for _, failure := range []string{"write", "close", "stat", "size"} {
		t.Run(failure, func(t *testing.T) {
			var attempt int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", `"v1"`)
				attempt++
				if failure == "stat" || failure == "size" {
					if attempt == 1 {
						w.Header().Set("Content-Length", "10")
						_, _ = io.WriteString(w, "0123")
						return
					}
					w.Header().Set("Content-Length", "6")
					w.Header().Set("Content-Range", "bytes 4-9/10")
					w.WriteHeader(http.StatusPartialContent)
					_, _ = io.WriteString(w, "456789")
					return
				}
				_, _ = io.WriteString(w, "data")
			}))
			defer srv.Close()
			testseam.Swap(t, &downloadOpenFile, func(name string, flags int, mode os.FileMode) (downloadFile, error) {
				f, err := os.OpenFile(name, flags, mode)
				if err != nil {
					return nil, err
				}
				return failingDownloadFile{File: f, failure: failure}, nil
			})
			_, err := DownloadWithConfig(t.Context(), srv.URL, filepath.Join(t.TempDir(), "out"), DefaultDownloadConfig())
			if err == nil || downloadCanFallback(err) {
				t.Fatalf("磁盘错误误归为网络错误: %v", err)
			}
		})
	}
}

func TestCrossPlatformCoverageDownloadHeaderIdleAndReadCancellation(t *testing.T) {
	for _, scenario := range []string{"header-idle", "read-cancel", "redirect", "weak-etag"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if scenario == "header-idle" {
					<-r.Context().Done()
					return
				}
				if scenario == "redirect" && r.URL.Path == "/" {
					http.Redirect(w, r, "/final", http.StatusFound)
					return
				}
				if scenario == "weak-etag" {
					w.Header().Set("ETag", `W/"v1"`)
				}
				if scenario == "read-cancel" {
					w.Header().Set("Content-Length", "10")
					_, _ = io.WriteString(w, "x")
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				_, _ = io.WriteString(w, "data")
			}))
			defer srv.Close()
			cfg := DefaultDownloadConfig()
			cfg.MaxRetries = 1
			cfg.IdleTimeout = 50 * time.Millisecond
			if scenario == "redirect" {
				cfg.CheckRedirect = func(*http.Request, []*http.Request) error { return nil }
			}
			if scenario == "read-cancel" {
				cfg.ProgressInterval = 0
				cfg.ProgressCallback = func(n, total int64) {
					if n > 0 {
						cancel()
					}
				}
			}
			_, err := DownloadWithConfig(ctx, srv.URL, filepath.Join(t.TempDir(), "out"), cfg)
			switch scenario {
			case "header-idle":
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("err=%v", err)
				}
			case "read-cancel":
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("err=%v", err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
