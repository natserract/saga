package saga

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/natserract/saga/internal/env"
	"github.com/natserract/saga/internal/redis_store"
	"github.com/stretchr/testify/assert"
)

func TestSaga_SuccessfulExecution(t *testing.T) {
	saga := NewSaga("TestSaga", nil)
	var requested, succeeded, failed int

	action := func() error {
		requested++
		succeeded++
		return nil
	}

	compensate := func() error {
		return nil
	}

	// Adding three successful actions
	for i := 0; i < 3; i++ {
		saga.AddAction(fmt.Sprintf("Action%d", i+1), action, compensate)
	}
	err := saga.Execute()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if requested != 3 || succeeded != 3 || failed != 0 {
		t.Fatalf("unexpected counters: requested=%d, succeeded=%d, failed=%d", requested, succeeded, failed)
	}

	fmt.Println("Successful execution counters matched!")
}

func TestSaga_ActionFails(t *testing.T) {
	saga := NewSaga("TestSaga", nil)
	var requested, succeeded, failed int

	action := func() error {
		requested++
		if requested == 2 { // Fail the second action
			failed++
			return errors.New("intentional fail")
		}
		succeeded++

		return nil
	}
	compensate := func() error {
		succeeded++
		return nil
	}
	for i := 0; i < 3; i++ {
		saga.AddAction(fmt.Sprintf("Action%d", i+1), action, compensate)
	}
	err := saga.Execute()

	if err != nil {
		t.Fatalf("Unexpected error during Execute with retries: %v", err)
	}

	if requested != 4 || succeeded != 3 || failed != 1 {
		t.Fatalf("unexpected counters: requested=%d, succeeded=%d, failed=%d", requested, succeeded, failed)
	}
}

func TestSaga_NextPrev(t *testing.T) {
	action1 := func() error {
		fmt.Println("Executing Action 1")
		return nil
	}
	compensation1 := func() error {
		fmt.Println("Compensating Action 1")
		return nil
	}

	action2 := func() error {
		fmt.Println("Executing Action 2")
		return errors.New("failure in action 2")
	}
	compensation2 := func() error {
		fmt.Println("Compensating Action 2")
		return nil
	}

	cfg := &SagaOptions{MaxRetries: 1, RetryWaitTime: 1 * time.Second}
	saga := NewSaga("TestSaga", cfg)
	saga.AddAction("Action1", action1, compensation1)
	saga.AddAction("Action2", action2, compensation2)

	err := saga.Next()
	if err != nil {
		t.Fatalf("Unexpected error during Next execution: %v", err)
	}
	assert.Equal(t, 1, saga.currentActionStep, "Current step should be 1")

	// Test the Prev method to compensate the actions
	err = saga.Prev()
	if err != nil {
		t.Fatalf("Unexpected error during Prev execution: %v", err)
	}
	assert.Equal(t, 0, saga.currentActionStep, "Current step should be 0")

	// Trying to call Prev again should result in an error as there's nothing left to compensate
	err = saga.Prev()
	if err == nil {
		t.Fatalf("Expected error during Prev execution on first action, but got none")
	}
}

func TestSaga_NextPrevWithRetries(t *testing.T) {
	// Define variables to track execution attempts
	action1Attempts := 0
	action2Attempts := 0

	action1 := func() error {
		action1Attempts++
		fmt.Println("Executing Action 1")
		return nil
	}
	compensation1 := func() error {
		fmt.Println("Compensating Action 1")
		return nil
	}

	action2 := func() error {
		action2Attempts++
		fmt.Println("Executing Action 2")
		if action2Attempts < 3 {
			return errors.New("temporary failure in action 2")
		}
		return nil
	}
	compensation2 := func() error {
		fmt.Println("Compensating Action 2")
		return nil
	}

	cfg := &SagaOptions{MaxRetries: 3, RetryWaitTime: 2 * time.Second}
	saga := NewSaga("TestSagaWithRetries", cfg)
	saga.AddAction("Action1", action1, compensation1)
	saga.AddAction("Action2", action2, compensation2)

	// Action 1
	err := saga.Next()
	if err != nil {
		t.Fatalf("Unexpected error during Next execution of second action with retries: %v", err)
	}

	// Action 2
	err = saga.Next()
	if err != nil {
		t.Fatalf("Unexpected error during Next execution of second action with retries: %v", err)
	}

	expectedRetries := 3 // Initial try + two retries before success
	if action2Attempts != expectedRetries {
		t.Fatalf("Expected action2 to be attempted %d times, but got %d", expectedRetries, action2Attempts)
	}

	// Test the Prev method to ensure compensation action 2
	err = saga.Prev()
	if err != nil {
		t.Fatalf("Unexpected error during Prev execution: %v", err)
	}
	assert.Equal(t, 1, saga.currentActionStep, "Current step should be 1")
}

func TestSaga_WithRedisStore(t *testing.T) {
	env.Load(".env")

	REDIS_URL := os.Getenv("REDIS_URL")
	assert.NotEmpty(t, REDIS_URL, "REDIS_URL should not be empty")

	t.Logf("Using Redis URL: %s", REDIS_URL)
	saga := NewSagaWithRedis(REDIS_URL, "TestSagaWithRedisStore", nil)

	var requested, succeeded int
	action := func() error {
		requested++
		succeeded++
		return nil
	}
	compensate := func() error {
		return nil
	}
	saga.AddAction("Action", action, compensate)

	err := saga.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check if key exists in Redis
	store, err := redis_store.NewRedisStore(&redis_store.Config{
		Url: REDIS_URL,
	})
	if err != nil {
		t.Fatalf("Unexpected error during Redis store creation: %v", err)
	}

	exists, err := store.Get(context.Background(), "TestSagaWithRedisStore-action-Action-1")
	if err != nil {
		t.Fatalf("Unexpected error during Redis key existence check: %v", err)
	}
	assert.NotNil(t, exists, "Key should exist in Redis")
}
