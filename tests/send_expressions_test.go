package tests

import (
	"encoding/json"
	"testing"

	"github.com/apple5343/golangProjectV2/tests/test"
	c "github.com/apple5343/grpc"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendExpression_HappyPath(t *testing.T) {
	ctx, st := test.New(t)
	name := gofakeit.Name()
	pass := randomPassword()

	respReg, err := st.AuthClient.Register(ctx, &c.RegisterRequest{
		Name:     name,
		Password: pass,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, respReg.GetUserId())

	respSend, err := st.CalcClient.AddTask(ctx, &c.AddTaskRequest{UserId: respReg.UserId, Task: "5+5"})
	require.NoError(t, err)
	assert.NotEmpty(t, respSend.Task)
	var task map[string]interface{}
	err = json.Unmarshal([]byte(respSend.Task), &task)
	require.NoError(t, err)
	taskId := task["id"].(float64)

	respTask, err := st.CalcClient.GetTask(ctx, &c.GetTaskRequest{UserId: respReg.UserId, TaskId: int64(taskId)})
	err = json.Unmarshal([]byte(respTask.Task), &task)
	require.NoError(t, err)
	require.True(t, task["expression"] == "5+5")

	respSend, err = st.CalcClient.AddTask(ctx, &c.AddTaskRequest{UserId: respReg.UserId, Task: "5+5"})
	require.NoError(t, err)
	assert.NotEmpty(t, respSend.Task)
	err = json.Unmarshal([]byte(respSend.Task), &task)
	require.NoError(t, err)

	respAllTasks, err := st.CalcClient.GetAllTasks(ctx, &c.GetAllTasksRequest{UserId: respReg.UserId})
	require.NoError(t, err)
	assert.NotEmpty(t, respAllTasks.Tasks)
	var tasks []map[string]interface{}
	err = json.Unmarshal([]byte(respAllTasks.Tasks), &tasks)
	require.NoError(t, err)
	require.True(t, len(tasks) == 2)
}

func TestAddTask_FailCases(t *testing.T) {
	ctx, st := test.New(t)

	name := gofakeit.Name()
	pass := randomPassword()
	respReg, err := st.AuthClient.Register(ctx, &c.RegisterRequest{
		Name:     name,
		Password: pass,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, respReg.GetUserId())

	tests := []struct {
		name          string
		uid           int64
		task          string
		expectedError string
	}{
		{
			name:          "Несуществующий пользователь",
			uid:           -1,
			task:          "5+5",
			expectedError: "user not found",
		},
		{
			name:          "недопустимое выражение",
			uid:           respReg.UserId,
			task:          "32/0",
			expectedError: "invalid expression",
		},
		{
			name:          "недопустимое выражение",
			uid:           respReg.UserId,
			task:          "32=0",
			expectedError: "invalid expression",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.CalcClient.AddTask(ctx, &c.AddTaskRequest{UserId: int64(tt.uid), Task: tt.task})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
