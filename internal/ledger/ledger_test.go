package ledger

import (
	"sync"
	"testing"
)

func TestSummarize(t *testing.T) {
	kinds := []string{"income", "expense", "income", "expense", "transfer"}
	amounts := []int64{10000, 3500, 2500, 1500, 99999}
	income, expense, balance := Summarize(kinds, amounts)
	if income != 12500 || expense != 5000 || balance != 7500 {
		t.Fatalf("got income=%d expense=%d balance=%d", income, expense, balance)
	}
}

func TestApplyTransfer(t *testing.T) {
	nf, nt, err := ApplyTransfer(1000, 500, 400)
	if err != nil || nf != 600 || nt != 900 {
		t.Fatalf("got %d %d %v", nf, nt, err)
	}
	if _, _, err := ApplyTransfer(100, 0, 0); err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
	if _, _, err := ApplyTransfer(100, 0, -5); err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
	if _, _, err := ApplyTransfer(100, 0, 101); err != ErrInsufficientFunds {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

// TestConcurrentTransfers usa o razão puro sob mutex: N transferências
// concorrentes preservam a soma total (invariante da transferência atômica).
func TestConcurrentTransfers(t *testing.T) {
	const n = 64
	var mu sync.Mutex
	var wg sync.WaitGroup
	from, to := int64(1_000_000), int64(0)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var amt int64 = 1000
			var f, tt int64
			if i%2 == 1 {
				// metade volta no sentido contrário para exercitar as duas direções
				mu.Lock()
				defer mu.Unlock()
				var err error
				tt, f, err = ApplyTransfer(to, from, amt)
				if err != nil {
					return // sem saldo: esperado em corrida, soma segue preservada
				}
				to, from = tt, f
				return
			}
			mu.Lock()
			defer mu.Unlock()
			var err error
			f, tt, err = ApplyTransfer(from, to, amt)
			if err != nil {
				t.Errorf("unexpected err: %v", err)
				return
			}
			from, to = f, tt
		}(i)
	}
	wg.Wait()
	if from+to != 1_000_000 {
		t.Fatalf("sum not preserved: from=%d to=%d", from, to)
	}
}

func TestLockOrder(t *testing.T) {
	a, b := LockOrder("zzz", "aaa")
	if a != "aaa" || b != "zzz" {
		t.Fatalf("got %q %q", a, b)
	}
}
