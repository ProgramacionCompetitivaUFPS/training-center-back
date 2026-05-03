[HIGH] rows.Err() returned raw from goroutine — context is swallowed at eg.Wait()

rows.Err() here returns the raw *pgconn.PgError or connection error directly from the goroutine. At the eg.Wait() call site the error is wrapped as apperror.NewInternal(), discarding the original value, error type, and all context. The operator log will show nothing about which goroutine failed or why.

The same issue exists on the memberships goroutine at the equivalent return rows.Err() below. Both goroutines also drop the slog on their scan-failure paths.

Fix by logging and wrapping before returning from each goroutine:

Suggested change
		return rows.Err()
		return rows.Err()
→ Replace both return rows.Err() calls and the un-logged scan errors with:

		if err := rows.Err(); err != nil {
			slog.ErrorContext(egCtx, "BulkStats count rows error", "error", err)
			return apperror.NewInternal()
		}
		return nil
and similarly for the memberships goroutine. Then at eg.Wait(), propagate directly (return nil, err) since the error is already wrapped.