package scheduler

import (
	"fmt"
	"sync"
)

const MaxPriority = 100

/* prioritzation discipline

   0 - ad hoc backup
       ad hoc restore
       ad hoc test-store
       ad hoc purge

   10 - ad hoc agent-status
   20 - scheduled backup
   30 - scheduled test-store
   40 - scheduled agent status
   50 - scheduled archive purge
*/

type Scheduler struct {
	lock    sync.Mutex
	workers []Worker
	chores  [][]Chore
}

func New(workers int) *Scheduler {
	pool := make([]Worker, workers)
	for i := range pool {
		pool[i] = NewWorker()
	}

	return &Scheduler{
		workers: pool,
		chores:  make([][]Chore, MaxPriority),
	}
}

func (s *Scheduler) RunChore(chore Chore) error {
	/* ... */
	return nil
}

func (s *Scheduler) Schedule(priority int, chore Chore) error {
	if priority < 1 || priority > MaxPriority {
		return fmt.Errorf("invalid task priority '%d'; must be between 1 (highest) and %d (lowest)", priority, MaxPriority)
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.chores[priority-1] = append(s.chores[priority-1], chore)
	return nil
}

func (s *Scheduler) Run() {
	prio := 0

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, worker := range s.workers {
		if !worker.Available() {
			continue
		}

		for len(s.chores[prio]) == 0 {
			prio += 1
			if prio == MaxPriority {
				return
			}
		}

		chore := s.chores[prio][0]
		s.chores[prio] = s.chores[prio][1:]

		worker.Execute(chore)
	}
}
