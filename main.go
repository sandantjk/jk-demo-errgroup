package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	srv0 := newServer(":8080", "server0")
	srv1 := newServer(":8081", "server1")
	srv2 := newServer(":8082", "server1")

	// 使用context控制生命周期
	srvGroup, groupCtx := errgroup.WithContext(context.Background())
	ctx, cancelFunc := context.WithCancel(groupCtx)

	srvGroup.Go(doSrv(ctx, cancelFunc, srv0))
	srvGroup.Go(doSrv(ctx, cancelFunc, srv1))
	srvGroup.Go(doSrv(ctx, cancelFunc, srv2))

	// 监听kill信号
	go waitKill(cancelFunc)

	err := srvGroup.Wait()
	if err != nil {
		log.Printf("App start fail -> %v", err)
	} else {
		log.Printf("App shutdown success")
	}
}

// 启动服务
func doSrv(ctx context.Context, cancelFunc context.CancelFunc, srv *http.Server) func() error {
	return func() error {
		var rtErr error
		go func() {
			// 启动服务，启动失败发起取消
			if err := srv.ListenAndServe(); err != nil {
				// 如果是服务已关闭，异常忽略
				if !errors.Is(err, http.ErrServerClosed) {
					rtErr = err
				}
				cancelFunc()
			}
		}()
		// 监听取消信号
		select {
		case <-ctx.Done():
			shutDownSrv(srv)
		}
		return rtErr
	}
}

// 停止指定服务。使用context控制超时
func shutDownSrv(srv *http.Server) {
	//log.Printf("[%s] shutdown...", srv.Addr)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFunc()
	_ = srv.Shutdown(ctx)
	log.Printf("[%s] shutdown success", srv.Addr)
}

// 监听kill信号
func waitKill(cancelFunc context.CancelFunc) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancelFunc()
}

// 模拟启用http server
func newServer(port, msg string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(msg))
	})
	srv := &http.Server{
		Addr:    port,
		Handler: mux,
	}
	return srv
}
