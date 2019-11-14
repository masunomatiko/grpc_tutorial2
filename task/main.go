package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/masunomatiko/grpc_tutorial2/shared/interceptor"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	pbActivity "github.com/masunomatiko/grpc_tutorial2/proto/activity"
	pbProject "github.com/masunomatiko/grpc_tutorial2/proto/project"
	pbTask "github.com/masunomatiko/grpc_tutorial2/proto/task"
	"google.golang.org/grpc"
)

const port = ":50051"

func main() {
	activityConn, err := grpc.Dial(
		os.Getenv("ACTIVITY_SERVICE_ADDR"),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to dial activity: %s", err)
	}
	projectConn, err := grpc.Dial(
		os.Getenv("PROJECT_SERVICE_ADDR"),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to dial project: %s", err)
	}

	chain := grpc_middleware.ChainUnaryServer(
		interceptor.XTraceID(),
		interceptor.Logging(),
		interceptor.XUserID(),
	)
	srvOpt := grpc.UnaryInterceptor(chain)
	srv := grpc.NewServer(srvOpt)

	pbTask.RegisterTaskServiceServer(
		srv,
		&TaskService{
			store:          NewStoreOnMemory(),
			activityClient: pbActivity.NewActivityServiceClient(activityConn),
			projectClient:  pbProject.NewProjectServiceClient(projectConn),
		},
	)

	go func() {
		listener, err := net.Listen("tcp", port)
		if err != nil {
			log.Fatalf("failed to create listener: %s", err)
		}
		log.Printf("start server on port %s\n", port)
		if err := srv.Serve(listener); err != nil {
			log.Printf("failed to exit serve: %s\n", err)
		}
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGTERM)
	<-sigint
	log.Println("received a signal of gracefull shutdown")
	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()
	ctx, cancel := context.WithTimeout(
		context.Background(),
		1*time.Minute,
	)
	select {
	case <-ctx.Done():
		srv.Stop()
	case <-stopped:
		cancel()
	}
	log.Println("completed graceful shutdown")
}
