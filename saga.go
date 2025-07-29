package saga

import (
	"fmt"
	"sync"
	"time"
)

type SagaStatus int

const (
	// Initial state before the transaction begins.
	Pending SagaStatus = iota

	// The transaction is currently being executed.
	InProgress

	// The transaction has successfully finished.
	Completed

	// The transaction failed to complete successfully
	Failed

	// Compensation actions are being executed to revert previously completed transactions
	Compensating

	// All necessary compensations have been completed successfully
	Compensated
)

type SagaAction struct {
	name       string
	action     func() error
	compensate func() error

	// Ensure that status is never accessed with a race-condition.
	statusLock sync.RWMutex
	status     SagaStatus
}

type Saga struct {
	Name    string
	Actions []SagaAction

	config           *SagaOptions
	idempotencyStore *IdempotencyStore

	currentActionStep int
}

func NewSaga(name string, cfg *SagaOptions) *Saga {
	if cfg == nil {
		cfg = &SagaOptions{
			MaxRetries:    defaultMaxRetries,
			RetryWaitTime: defaultRetryWaitTime,
		}
	}

	return &Saga{
		Name:              name,
		Actions:           []SagaAction{},
		config:            cfg,
		idempotencyStore:  NewIdempotencyStore(),
		currentActionStep: 0,
	}
}

func (s *Saga) AddAction(
	name string,
	action func() error,
	compensate func() error,
) {
	s.Actions = append(s.Actions, SagaAction{
		name:       name,
		action:     action,
		compensate: compensate,
		status:     Pending,
	})
}

// Execute runs all actions in the saga, handling failures and triggering rollbacks if needed
func (s *Saga) Execute() error {
	for i := range s.Actions {
		action := &s.Actions[i]

		idempotencyKey := fmt.Sprintf("%s-action-%s-%d", s.Name, action.name, i+1)
		if s.idempotencyStore.IsCompleted(idempotencyKey) {
			continue
		}

		err := action.executeWithRetry(s.config.MaxRetries, s.config.RetryWaitTime)
		if err != nil {
			action.updateStatus(Failed)
			rollback(s.Actions[:i+1]) // Rollback all previous actions
			return err
		}
		action.updateStatus(Completed)
		s.idempotencyStore.MarkCompleted(idempotencyKey)
	}
	return nil
}

func (s *Saga) Next() error {
	if s.currentActionStep >= len(s.Actions) {
		return fmt.Errorf("no more actions to execute in saga %s", s.Name)
	}

	action := &s.Actions[s.currentActionStep]
	idempotencyKey := fmt.Sprintf("%s-action-%s-%d", s.Name, action.name, s.currentActionStep+1)

	if s.idempotencyStore.IsCompleted(idempotencyKey) {
		s.currentActionStep++
		return s.Next() // move to the next action
	}

	err := action.executeWithRetry(s.config.MaxRetries, s.config.RetryWaitTime)
	if err != nil {
		action.updateStatus(Failed)
		rollback(s.Actions[:s.currentActionStep]) // Rollback all previous actions
		return err
	}

	action.updateStatus(Completed)
	s.idempotencyStore.MarkCompleted(idempotencyKey)
	s.currentActionStep++
	if s.currentActionStep == len(s.Actions) {
		return nil
	}
	return nil
}

func (s *Saga) Prev() error {
	if s.currentActionStep == 0 {
		return fmt.Errorf("No previous action to rollback in saga %s", s.Name)
	}

	s.currentActionStep--
	action := &s.Actions[s.currentActionStep]

	if action.getStatus() == Completed {
		action.updateStatus(Compensating)
		err := action.compensate()
		if err != nil {
			return err
		}
		action.updateStatus(Compensated)
	}
	return nil
}

func (a *SagaAction) executeWithRetry(maxRetries int, retryWaitTime time.Duration) error {
	var err error

	for _ = range maxRetries {
		a.updateStatus(InProgress)
		err = a.action()

		if err == nil {
			return nil
		}
		time.Sleep(retryWaitTime) // Wait before retrying
	}
	return err
}

// rollback compensates each completed action in reverse order
func rollback(actions []SagaAction) {
	// Compensate in reverse order
	for i := len(actions) - 1; i >= 0; i-- {
		action := &actions[i]
		if action.getStatus() == Failed {
			action.updateStatus(Compensating)
			err := action.compensate()
			if err == nil {
				action.updateStatus(Compensated)
			} else {
				continue
			}
		}
	}
}

func (a *SagaAction) updateStatus(newStatus SagaStatus) {
	a.statusLock.Lock()
	defer a.statusLock.Unlock()
	a.status = newStatus
}

func (a *SagaAction) getStatus() SagaStatus {
	a.statusLock.RLock()
	defer a.statusLock.RUnlock()
	return a.status
}
