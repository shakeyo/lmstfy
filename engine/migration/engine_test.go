package migration

import (
	"bytes"
	"testing"
	"time"
)

func TestEngine_Publish(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 1")
	jobID, err := e.Publish("ns-engine", "q1", body, 10, 2, 1)
	t.Log(jobID)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}

	// Publish no-delay job
	jobID, err = e.Publish("ns-engine", "q1", body, 10, 0, 1)
	t.Log(jobID)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}
	// Make sure the new engine received the job
	job, err := NewRedisEngine.Consume("ns-engine", "q1", 3, 0)
	if job.ID() != jobID {
		t.Fatal("NewRedisEngine should received the job")
	}
}

func TestEngine_Consume(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 2")
	jobID, err := e.Publish("ns-engine", "q2", body, 10, 2, 1)
	t.Log(jobID)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}
	job, err := e.Consume("ns-engine", "q2", 3, 3)
	if err != nil {
		t.Fatalf("Failed to consume: %s", err)
	}
	if !bytes.Equal(body, job.Body()) || jobID != job.ID() {
		t.Fatalf("Mistmatched job data")
	}

	// Consume job that's published in no-delay way
	jobID, err = e.Publish("ns-engine", "q2", body, 10, 0, 1)
	t.Log(jobID)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}
	job, err = e.Consume("ns-engine", "q2", 3, 0)
	if err != nil {
		t.Fatalf("Failed to consume: %s", err)
	}
	if !bytes.Equal(body, job.Body()) || jobID != job.ID() {
		t.Fatalf("Mistmatched job data")
	}
}

// Consume the first one from multi publish
func TestEngine_Consume2(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 3")
	_, err := e.Publish("ns-engine", "q3", []byte("delay msg"), 10, 5, 1)
	jobID, err := e.Publish("ns-engine", "q3", body, 10, 2, 1)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}
	job, err := e.Consume("ns-engine", "q3", 3, 3)
	if err != nil {
		t.Fatalf("Failed to consume: %s", err)
	}
	if !bytes.Equal(body, job.Body()) || jobID != job.ID() {
		t.Fatalf("Mistmatched job data")
	}
}

func TestEngine_ConsumeMulti(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 4")
	jobID, err := e.Publish("ns-engine", "q4", body, 10, 3, 1)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}
	jobID2, err := e.Publish("ns-engine", "q5", body, 10, 1, 1)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}

	job2, err := e.ConsumeMulti("ns-engine", []string{"q4", "q5"}, 5, 5)
	if err != nil {
		t.Fatalf("Failed to consume from multiple queues: %s", err)
	}
	if job2.Queue() != "q5" || job2.ID() != jobID2 { // q5's job should be fired first
		t.Error("Mismatched job data")
	}

	job1, err := e.ConsumeMulti("ns-engine", []string{"q4", "q5"}, 5, 5)
	if err != nil {
		t.Fatalf("Failed to consume from multiple queues: %s", err)
	}
	if job1.Queue() != "q4" || job1.ID() != jobID { // q4's job should be fired next
		t.Fatalf("Failed to consume from multiple queues: %s", err)
	}
}

func TestEngine_Peek(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 6")
	jobID, err := e.Publish("ns-engine", "q6", body, 10, 0, 1)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}
	job, err := e.Peek("ns-engine", "q6", "")
	if job.ID() != jobID || !bytes.Equal(job.Body(), body) {
		t.Fatal("Mismatched job")
	}
}

func TestEngine_DrainOld(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 7")
	jobID, err := OldRedisEngine.Publish("ns-engine", "q7", body, 10, 0, 1)
	job, err := e.Consume("ns-engine", "q7", 5, 0)
	if err != nil {
		t.Fatal("Failed to drain the old engine's data")
	}
	if job.ID() != jobID {
		t.Fatal("Mismatched job")
	}
}

func TestEngine_BatchConsume(t *testing.T) {
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	body := []byte("hello msg 8")
	jobID, err := e.Publish("ns-engine", "q8", body, 10, 2, 1)
	if err != nil {
		t.Fatalf("Failed to publish: %s", err)
	}

	jobs, err := e.BatchConsume("ns-engine", "q8", 2, 5, 0)
	if err != nil {
		t.Fatalf("Failed to Batch consume: %s", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("Wrong job consumed")
	}

	jobs, err = e.BatchConsume("ns-engine", "q8", 2, 5, 3)
	if err != nil {
		t.Fatalf("Failed to Batch consume: %s", err)
	}
	if len(jobs) != 1 || !bytes.Equal(body, jobs[0].Body()) || jobID != jobs[0].ID() {
		t.Fatalf("Mistmatched job data")
	}

	// Consume some jobs
	jobIDMap := map[string]bool{}
	for i := 0; i < 4; i++ {
		jobID, err := e.Publish("ns-engine", "q8", body, 10, 0, 1)
		t.Log(jobID)
		if err != nil {
			t.Fatalf("Failed to publish: %s", err)
		}
		jobIDMap[jobID] = true
	}

	// First time batch consume three jobs
	jobs, err = e.BatchConsume("ns-engine", "q8", 3, 3, 3)
	if err != nil {
		t.Fatalf("Failed to consume: %s", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("Mistmatched jobs count")
	}
	for _, job := range jobs {
		if !bytes.Equal(body, job.Body()) || !jobIDMap[job.ID()] {
			t.Fatalf("Mistmatched job data")
		}
	}

	// Second time batch consume can only get a single job
	jobs, err = e.BatchConsume("ns-engine", "q8", 3, 3, 3)
	if err != nil {
		t.Fatalf("Failed to consume: %s", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("Mistmatched jobs count")
	}
	if !bytes.Equal(body, jobs[0].Body()) || !jobIDMap[jobs[0].ID()] {
		t.Fatalf("Mistmatched job data")
	}

	jobs, err = e.BatchConsume("ns-engine", "q8", 3, 3, 3)
	if err != nil {
		t.Fatalf("Failed to consume: %s", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("Mistmatched jobs count")
	}
}

func TestEngine_DeadLetter_Size(t *testing.T) {
	body := []byte("hello msg 9")
	jobID, err := OldRedisEngine.Publish("ns-engine", "q9", body, 10, 0, 1)
	job, err := OldRedisEngine.Consume("ns-engine", "q9", 0, 0)
	if err != nil {
		t.Fatal("Failed to drain the old engine's data")
	}
	if job.ID() != jobID {
		t.Fatal("Mismatched job")
	}
	jobID, err = NewRedisEngine.Publish("ns-engine", "q9", body, 10, 0, 1)
	job, err = NewRedisEngine.Consume("ns-engine", "q9", 0, 0)
	if job.ID() != jobID {
		t.Fatal("Mismatched job")
	}
	time.Sleep(2 * time.Second)
	e := NewEngine(OldRedisEngine, NewRedisEngine)
	size, _ := e.SizeOfDeadLetter("ns-engine", "q9")
	if size != 2 {
		t.Fatalf("Expected the deadletter queue size is: %d, but got %d\n", 2, size)
	}
}
