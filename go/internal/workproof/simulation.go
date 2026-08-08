// Package workproof contains a deterministic, dependency-light model of the
// P0 escrow and verdict checks. It is intentionally not production custody
// code; it is the pre-build lab used to exercise safety properties before
// wiring the same rules into Solidity.
package workproof

import (
	"errors"
	"fmt"
	"math/big"
)

type State uint8

const (
	Created State = iota
	Accepted
	AwaitingResubmission
	Retryable
	Paid
	Cancelled
	Refunded
)

type Outcome uint8

const (
	Pass Outcome = iota + 1
	Fail
	Inconclusive
)

type Verdict struct {
	Escrow, JobID, InstructionID string
	Attempt                      uint64
	Outcome                      Outcome
	Passed, Executed             uint32
	IssuedAt, ExpiresAt          uint64
	TEEID                        string
}

type Job struct {
	Client, Contractor, ExpectedTEE string
	Principal, Fee                  *big.Int
	State                           State
	Attempt                         uint64
	SubmitBy, GraceEnds             uint64
	Settled                         bool
}

type Ledger struct{ Client, Contractor, Treasury *big.Int }

func NewJob(client, contractor, tee string, principal, fee *big.Int, submitBy, graceEnds uint64) (*Job, *Ledger) {
	return &Job{Client: client, Contractor: contractor, ExpectedTEE: tee, Principal: new(big.Int).Set(principal), Fee: new(big.Int).Set(fee), State: Created, SubmitBy: submitBy, GraceEnds: graceEnds}, &Ledger{Client: new(big.Int), Contractor: new(big.Int), Treasury: new(big.Int)}
}

func (j *Job) Accept() error {
	if j.State != Created {
		return errors.New("job is not awaiting acceptance")
	}
	j.State = Accepted
	return nil
}

func (j *Job) Submit(now uint64) error {
	if j.State != Accepted && j.State != AwaitingResubmission && j.State != Retryable {
		return errors.New("job is not submit-ready")
	}
	if now > j.SubmitBy {
		return errors.New("submission deadline passed")
	}
	j.Attempt++
	return nil
}

func (j *Job) Settle(now uint64, v Verdict, ledger *Ledger) error {
	if j.Settled {
		return errors.New("job already settled")
	}
	if v.Outcome == Pass {
		if v.TEEID != j.ExpectedTEE {
			return errors.New("unexpected TEE")
		}
		if v.Attempt != j.Attempt {
			return errors.New("stale attempt")
		}
		if v.ExpiresAt < now {
			return errors.New("verdict expired")
		}
		j.State, j.Settled = Paid, true
		ledger.Contractor.Add(ledger.Contractor, j.Principal)
		ledger.Treasury.Add(ledger.Treasury, j.Fee)
		return nil
	}
	if v.Outcome == Fail {
		j.State = AwaitingResubmission
		return nil
	}
	if v.Outcome == Inconclusive {
		j.State = Retryable
		return nil
	}
	return fmt.Errorf("unknown outcome %d", v.Outcome)
}

func (j *Job) Cancel(now uint64, ledger *Ledger) error {
	if j.State != Created || now > j.SubmitBy {
		return errors.New("cannot cancel after acceptance/deadline")
	}
	j.State, j.Settled = Cancelled, true
	ledger.Client.Add(ledger.Client, new(big.Int).Add(j.Principal, j.Fee))
	return nil
}

func (j *Job) Refund(now uint64, ledger *Ledger) error {
	if j.Settled || now <= j.GraceEnds {
		return errors.New("refund not available")
	}
	j.State, j.Settled = Refunded, true
	ledger.Client.Add(ledger.Client, new(big.Int).Add(j.Principal, j.Fee))
	return nil
}

// DeterministicRound models secure future randomness; callers cannot advance
// it without the explicitly supplied future round.
func DeterministicRound(seed uint64, futureRound uint64) uint64 {
	return seed ^ (futureRound * 0x9e3779b97f4a7c15)
}
