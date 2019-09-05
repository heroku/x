package tickgroup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	group := New(ctx)

	var ticks uint32
	group.Go(5*time.Millisecond, func() error {
		if atomic.AddUint32(&ticks, 1) == 5 {
			cancel()
			return nil
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}

	if want, got := 5, int(atomic.LoadUint32(&ticks)); want != got {
		t.Fatalf("want tick count %d, got %d", want, got)
	}
}

func TestGroupWithContext(t *testing.T) {
	group, ctx := WithContext(context.Background())

	wantErr := errors.New("done")

	var ticks uint32
	group.Go(5*time.Millisecond, func() error {
		if atomic.AddUint32(&ticks, 1) == 5 {
			return wantErr
		}
		return nil
	})

	if gotErr := group.Wait(); gotErr != wantErr {
		t.Fatalf("want err: %v, got err %v", wantErr, gotErr)
	}

	select {
	case <-ctx.Done():
	default:
		t.Fatal("wanted ctx to have been canceled")
	}

	if want, got := 5, int(atomic.LoadUint32(&ticks)); want != got {
		t.Fatalf("want tick count %d, got %d", want, got)
	}
}

func TestGroupRunsImmediatelyBeforeWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	group := New(ctx)

	var ticks uint32
	group.Go(time.Hour, func() error {
		if atomic.AddUint32(&ticks, 1) == 1 {
			cancel()
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}

	if want, got := 1, int(atomic.LoadUint32(&ticks)); want != got {
		t.Fatalf("want tick count %d, got %d", want, got)
	}
}
