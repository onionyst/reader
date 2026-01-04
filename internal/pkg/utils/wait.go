package utils

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

func Wait(services []string, timeout int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if err := waitContext(ctx, services); err != nil {
		panic(fmt.Errorf("dependency services failed: %w", err))
	}
}

func waitContext(ctx context.Context, services []string) error {
	if len(services) == 0 {
		return nil
	}

	dialer := &net.Dialer{
		Timeout: 2 * time.Second,
	}
	g, ctx := errgroup.WithContext(ctx)

	for _, srv := range services {
		g.Go(func() error {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				conn, err := dialer.DialContext(ctx, "tcp", srv)
				if err == nil {
					_ = conn.Close()
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
		})
	}

	return g.Wait()
}
