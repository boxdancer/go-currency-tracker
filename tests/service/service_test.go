package service_test

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/boxdancer/go-currency-tracker/internal/currency"
    "github.com/boxdancer/go-currency-tracker/tests/testutil"
)

func mustHaveIDs(t *testing.T, got map[string]map[string]float64, ids ...string) {
    t.Helper()
    for _, id := range ids {
        if _, ok := got[id]; !ok {
            t.Fatalf("expected id %q in results", id)
        }
    }
}
func mustNotHaveIDs(t *testing.T, got map[string]map[string]float64, ids ...string) {
    t.Helper()
    for _, id := range ids {
        if _, ok := got[id]; ok {
            t.Fatalf("did not expect id %q in results", id)
        }
    }
}

// helper to copy a map[string]string (to avoid accidental sharing between subtests)
func copyPairs(in map[string]string) map[string]string {
    out := make(map[string]string, len(in))
    for k, v := range in {
        out[k] = v
    }
    return out
}

func TestService_GetMany(t *testing.T) {
    basePairs := map[string]string{
        "bitcoin":  "usd",
        "ethereum": "usd",
        "usd":      "rub",
    }

    type fields struct {
        fake *testutil.FakePriceClient
    }
    type args struct {
        // ctxFactory создаёт контекст и возвращает cancel, чтобы мы могли defer cancel() внутри t.Run
        ctxFactory func() (context.Context, context.CancelFunc)
        pairs      map[string]string
    }

    tests := []struct {
        name       string
        fields     fields
        args       args
        wantErr    bool
        mustHave   []string
        mustAbsent []string
    }{
        {
            name: "all success",
            fields: fields{fake: &testutil.FakePriceClient{
                Responses: map[testutil.Key]float64{
                    {ID: "bitcoin", VS: "usd"}:  100.0,
                    {ID: "ethereum", VS: "usd"}: 10.0,
                    {ID: "usd", VS: "rub"}:      70.5,
                },
            }},
            args: args{
                ctxFactory: func() (context.Context, context.CancelFunc) {
                    // обычный background — не нужно отменять, но возвращаем cancel чтобы API единообразно использовался
                    return context.Background(), func() {}
                },
                pairs: copyPairs(basePairs),
            },
            wantErr:  false,
            mustHave: []string{"bitcoin", "ethereum", "usd"},
        },
        {
            name: "partial error",
            fields: fields{fake: &testutil.FakePriceClient{
                Responses: map[testutil.Key]float64{
                    {ID: "bitcoin", VS: "usd"}:  100.0,
                    {ID: "ethereum", VS: "usd"}: 10.0,
                },
                Errors: map[testutil.Key]error{
                    {ID: "usd", VS: "rub"}: testutil.Err("provider failure"),
                },
            }},
            args: args{
                ctxFactory: func() (context.Context, context.CancelFunc) {
                    return context.Background(), func() {}
                },
                pairs: copyPairs(basePairs),
            },
            wantErr:    true,
            mustHave:   []string{"bitcoin", "ethereum"},
            mustAbsent: []string{"usd"},
        },
        {
            name: "context cancelled",
            fields: fields{fake: &testutil.FakePriceClient{
                Responses: map[testutil.Key]float64{
                    {ID: "bitcoin", VS: "usd"}:  1,
                    {ID: "ethereum", VS: "usd"}: 1,
                },
                Delay: 150 * time.Millisecond,
            }},
            args: args{
                ctxFactory: func() (context.Context, context.CancelFunc) {
                    return context.WithTimeout(context.Background(), 50*time.Millisecond)
                },
                pairs: map[string]string{
                    "bitcoin":  "usd",
                    "ethereum": "usd",
                },
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        tt := tt // захватываем переменную на итерацию (без этого — проблемы при параллельном запуске)
        t.Run(tt.name, func(t *testing.T) {
            // создаём контекст именно внутри субтеста и откладываем cancel() в defer
            ctx, cancel := tt.args.ctxFactory()
            defer cancel()

            svc := currency.NewService(tt.fields.fake)
            got, err := svc.GetMany(ctx, tt.args.pairs)

            if tt.wantErr && err == nil {
                t.Fatalf("expected error, got nil")
            }
            if !tt.wantErr && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if tt.name == "context cancelled" {
                // мягкая проверка: ожидаем ошибку, связанную с контекстом
                if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
                    t.Fatalf("expected context-related error, got: %v", err)
                }
            }

            if len(tt.mustHave) > 0 {
                mustHaveIDs(t, got, tt.mustHave...)
            }
            if len(tt.mustAbsent) > 0 {
                mustNotHaveIDs(t, got, tt.mustAbsent...)
            }
        })
    }
}
