package main

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	pbActivity "github.com/masunomatiko/grpc_tutorial2/proto/activity"
	pbProject "github.com/masunomatiko/grpc_tutorial2/proto/project"
	pbTask "github.com/masunomatiko/grpc_tutorial2/proto/task"
	"github.com/masunomatiko/grpc_tutorial2/shared/md"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TaskService struct {
	store          Store
	activityClient pbActivity.ActivityServiceClient
	projectClient  pbProject.ProjectServiceClient
}

func (s *TaskService) CreateTask(
	ctx context.Context,
	req *pbTask.CreateTaskRequest,
) (*pbTask.CreateTaskResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"empty task name",
		)
	}
	resp, err := s.projectClient.FindProject(ctx,
		&pbProject.FindProjectRequest{
			ProjectId: req.GetProjectId(),
		})
	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			err.Error(),
		)
	}
	userID := md.GetUserIDFromContext(ctx)
	now := ptypes.TimestampNow()
	task, err := s.store.CreateTask(
		&pbTask.Task{
			Name:      req.GetName(),
			Status:    pbTask.Status_WAITING,
			ProjectId: resp.Project.GetId(),
			UserId:    userID,
			CreatedAt: now,
			UpdatedAt: now,
		},
	)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			err.Error(),
		)
	}
	content := &pbActivity.CreateTaskContent{
		TaskId:   task.GetTaskId(),
		TaskName: task.GetTaskName(),
	}
	any, err := ptypes.MarshalAny(content)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			err.Error(),
		)
	}
	if _, err := s.activityClient.CreateActivity(
		ctx,
		&pbActivity.CreateActivityRequest{
			Content: any,
		},
	); err != nil {
		return nil, err
	}
	return &pbTask.CreateTaskResponse{Task: task}, nil
}

func (s *TaskService) FindTasks(
	ctx context.Context,
	_ *empty.Empty,
) (*pbTask.FindTasksResponse, error) {
	userID := md.GetUserIDFromContext(ctx)
	tasks, err := s.store.FindTasks(userID)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			err.Error(),
		)
	}
	return &pbTask.FindProjectTasksResponse{Tasks: tasks}, nil
}

func (s *TaskService) FindProjectTasks(
	ctx context.Context,
	req *pbProject.FindProjectTasksRequest,
) (*pbTask.FindProjectTasksResponse, error) {
	userID := md.GetUserIDFromContext(ctx)
	tasks, err := s.store.FindProjectTasks(req.GetProjectId(), userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pbTask.FindProjectTasksResponse{Tasks: tasks}, nil
}

func (s *TaskService) UpdateTask(
	ctx context.Context,
	req *pbTask.UpdateTaskRequest,
) (*pbTask.UpdateTaskResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"empty task name",
		)
	}
	if req.GetStatus() == pbTask.Status_UNKNOWN {
		return nil, status.Error(
			codes.InvalidArgument,
			"unknown task name",
		)
	}
	userID := md.GetUserIDFromContext(ctx)
	task, err := s.store.FindTask(
		req.GetTaskId,
		userID,
	)
	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			err.Error(),
		)
	}
	updatedTask, err := s.store.UpdateTask(
		&pbTask.Task{
			Id:        task.Id,
			Name:      req.GetName(),
			Status:    req.GetStatus(),
			ProjectId: task.GetProjectId(),
			UserId:    task.GetUserId(),
			CreatedAt: task.GetCreatedAt(),
			UpdatedAt: ptypes.TimestampNow(),
		},
	)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			err.Error(),
		)
	}
	if task.GetStatus() == updatedTask.GetStatus() {
		return &pbTask.UpdatedTaskResponse{Task: updatedTask}, nil
	}
	any, err := ptypes.MarshalAny(&pbActivity.UpdateTaskStatusContent{
		TaskId:     updatedTask.GetId(),
		TaskName:   updatedTask.GetName(),
		TaskStatus: updatedTask.GetStatus(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _err := s.activityClient.CreateActivity(
		ctx,
		&pbActivity.CreateActivityRequest{
			Content: any,
		},
	); err != nil {
		return nil, err
	}
	return &pbTask.UpdateTaskResponse{Task: updatedTask}, nil
}
