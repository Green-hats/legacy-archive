package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"ani-rss/internal/bgm"
	"ani-rss/internal/cache"
	"ani-rss/internal/config"
	"ani-rss/internal/download"
	"ani-rss/internal/log"
	"ani-rss/internal/model"
	"ani-rss/internal/notify"
	"ani-rss/internal/rename"
	"ani-rss/internal/scrape"
	"ani-rss/internal/server"
	"ani-rss/internal/service"
	"ani-rss/internal/task"
	"ani-rss/internal/tmdb"
	"ani-rss/internal/util"
)

func main() {
	if err := config.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "加载配置失败:", err)
		os.Exit(1)
	}
	if err := log.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "初始化日志失败:", err)
	}
	log.SetMaxSize(config.Get().LogsMax)
	wireHooks()

	port := os.Getenv("PORT")
	if port == "" {
		port = "7789"
	}
	cfg := config.Get()
	_ = cfg

	task.Start()

	// wire stop/restart control from the /api/stop endpoint
	server.SetStopFn(func(shutdown bool) {
		go func() {
			time.Sleep(time.Second)
			task.Stop()
			log.Close()
			if shutdown {
				os.Exit(0)
			}
			// restart: re-exec ourselves
			exe, err := os.Executable()
			if err != nil {
				os.Exit(0)
			}
			cmd := exec.Command(exe, os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			_ = cmd.Start()
			os.Exit(0)
		}()
	})

	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: server.New(),
	}

	go func() {
		log.Infof("main", "ANI-RSS 服务已启动, 监听端口 %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("main", "HTTP 服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("main", "正在关闭服务...")
	task.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Close()
}

// wireHooks connects cross-package function hooks.
func wireHooks() {
	rename.BgmSubjectId = func(ani *model.Ani) string {
		return bgm.GetSubjectId(ani)
	}
	rename.JpTitle = func(ani *model.Ani) string {
		if ani.JpTitle != "" {
			return ani.JpTitle
		}
		subjectId := bgm.GetSubjectId(ani)
		if subjectId == "" {
			return ""
		}
		if info, err := bgm.GetBgmInfo(subjectId); err == nil && info != nil {
			ani.JpTitle = info.Name
			return info.Name
		}
		return ""
	}
	rename.TmdbEpisodeTitle = func(ani *model.Ani, ep int) (string, bool) {
		m := tmdb.GetEpisodeTitleMap(ani)
		name, ok := m[ep]
		return name, ok && name != ""
	}
	rename.BgmEpisodeTitle = func(ani *model.Ani, ep int) (string, string, bool) {
		cn, jp := bgm.GetEpisodeTitleMap(ani)
		nameCn := cn[ep]
		name := jp[ep]
		return nameCn, name, nameCn != "" || name != ""
	}

	notify.CurrentConfig = func() *model.Config { return config.Get() }
	notify.LogMsg = func(msg string) { log.Info("notification", msg) }
	notify.FileMoveFn = service.FileMove
	notify.OpenListUploadFn = service.OpenListUpload

	download.SetFindAniHook(service.FindAniByDownloadPath)
	service.ScrapeFn = scrape.Scrape
	service.RestartTasks = task.Restart
	service.ClearInMemoryCache = func() { cache.Default.Clear() }
	service.SetCacheSizeFunc(func() int { return cache.Default.Size() })
	util.SetLogWarn(func(logger, msg string) { log.Warn(logger, msg) })
	bgm.SaveCoverFn = service.SaveCover
	scrape.SetBgmSubjectIdHook(func(ani *model.Ani) string { return bgm.GetSubjectId(ani) })
}