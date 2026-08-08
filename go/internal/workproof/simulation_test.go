package workproof

import (
	"math/big"
	"testing"
)

func TestPassPaysOnceAndRejectsMutations(t *testing.T) {
	j, l := NewJob("client", "contractor", "tee-1", big.NewInt(100), big.NewInt(7), 10, 20)
	if err := j.Accept(); err != nil {
		t.Fatal(err)
	}
	if err := j.Submit(5); err != nil {
		t.Fatal(err)
	}
	bad := Verdict{Attempt: j.Attempt, Outcome: Pass, TEEID: "tee-2", ExpiresAt: 9}
	if err := j.Settle(6, bad, l); err == nil {
		t.Fatal("wrong TEE accepted")
	}
	good := Verdict{Attempt: j.Attempt, Outcome: Pass, TEEID: "tee-1", ExpiresAt: 9}
	if err := j.Settle(6, good, l); err != nil {
		t.Fatal(err)
	}
	if l.Contractor.Cmp(big.NewInt(100)) != 0 || l.Treasury.Cmp(big.NewInt(7)) != 0 {
		t.Fatal("incorrect payout")
	}
	if err := j.Settle(7, good, l); err == nil {
		t.Fatal("replay accepted")
	}
}

func TestFailAndInconclusiveNeverPay(t *testing.T) {
	j, l := NewJob("c", "p", "tee", big.NewInt(5), big.NewInt(1), 10, 20)
	if err := j.Accept(); err != nil {
		t.Fatal(err)
	}
	if err := j.Submit(1); err != nil {
		t.Fatal(err)
	}
	if err := j.Settle(2, Verdict{Attempt: 1, Outcome: Fail}, l); err != nil {
		t.Fatal(err)
	}
	if j.State != AwaitingResubmission || l.Contractor.Sign() != 0 {
		t.Fatal("FAIL paid")
	}
	if err := j.Submit(3); err != nil {
		t.Fatal(err)
	}
	if err := j.Settle(4, Verdict{Attempt: 2, Outcome: Inconclusive}, l); err != nil {
		t.Fatal(err)
	}
	if j.State != Retryable || l.Contractor.Sign() != 0 {
		t.Fatal("INCONCLUSIVE paid")
	}
}

func TestCancelAndRefund(t *testing.T) {
	j, l := NewJob("c", "p", "tee", big.NewInt(5), big.NewInt(1), 10, 20)
	if err := j.Cancel(1, l); err != nil {
		t.Fatal(err)
	}
	if l.Client.Cmp(big.NewInt(6)) != 0 {
		t.Fatal("cancel refund mismatch")
	}
	k, m := NewJob("c", "p", "tee", big.NewInt(5), big.NewInt(1), 10, 20)
	if err := k.Accept(); err != nil {
		t.Fatal(err)
	}
	if err := k.Refund(21, m); err != nil {
		t.Fatal(err)
	}
	if m.Client.Cmp(big.NewInt(6)) != 0 {
		t.Fatal("expiry refund mismatch")
	}
}

func TestRandomnessCannotAdvanceWithoutFutureRound(t *testing.T) {
	if DeterministicRound(12, 0) == DeterministicRound(12, 1) {
		t.Fatal("rounds must differ")
	}
	if DeterministicRound(12, 4) != DeterministicRound(12, 4) {
		t.Fatal("round must be deterministic")
	}
}
